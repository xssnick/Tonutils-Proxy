package proxy

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	tunnelConfig "github.com/ton-blockchain/adnl-tunnel/config"
	"github.com/ton-blockchain/adnl-tunnel/tunnel"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/adnl"
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/dns"
	"github.com/xssnick/tonutils-proxy/proxy/transport"
	"github.com/xssnick/tonutils-storage/config"
	"github.com/xssnick/tonutils-storage/storage"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

type proxy struct {
	version   string
	blockHttp bool
}

var client *http.Client

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	if req.URL.Scheme == "" {
		// if no scheme - we check forwarded proto
		req.URL.Scheme = req.Header.Get("X-Forwarded-Proto")
	}

	if req.Method == "CONNECT" {
		wr.WriteHeader(http.StatusOK)
		return
	}

	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		msg := "unsupported protocol scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		return
	}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	delHopHeaders(req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}
	req.Header.Set("X-Tonutils-Proxy", p.version)

	var c = http.DefaultClient
	if strings.HasSuffix(req.Host, ".ton") || strings.HasSuffix(req.Host, ".adnl") ||
		strings.HasSuffix(req.Host, ".t.me") || strings.HasSuffix(req.Host, ".bag") {
		log.Debug().Str("method", req.Method).Str("url", req.URL.String()).Msg("over rldp")
		// proxy requests to ton using special client
		c = client
	} else {
		if p.blockHttp {
			http.Error(wr, "HTTP Not allowed", http.StatusBadRequest)
			return
		}

		log.Debug().Str("method", req.Method).Str("url", req.URL.String()).Msg("over http")
	}

	resp, err := c.Do(req)
	if err != nil {
		text := err.Error()
		if strings.Contains(text, "context deadline exceeded") {
			http.Error(wr, "TON Site "+req.URL.Host+" is not responding.", http.StatusBadGateway)
		} else {
			http.Error(wr, "RLDP Proxy Error:\n"+text, http.StatusBadGateway)
		}
		log.Warn().Str("err", text).Str("method", req.Method).Str("url", req.URL.String()).Msg("cannot open")
		return
	}
	defer resp.Body.Close()

	log.Debug().Str("status", resp.Status).Str("addr", req.RemoteAddr).Msg("loading")

	delHopHeaders(resp.Header)

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

type State struct {
	Type    string
	State   string
	Stopped bool
}

type Proxy struct {
	httpServer  *http.Server
	dht         *dht.Client
	tr          *transport.Transport
	st          *storage.Server
	gateProxy   *adnl.Gateway
	gateStorage *adnl.Gateway
	connPool    *liteclient.ConnectionPool
}

func StartProxy(addr string, verbosity int, res chan<- State, versionAndDevice string, blockHttp bool, netConfigPath, tunnelConfigPath string) (*Proxy, error) {
	if res != nil {
		res <- State{
			Type:  "loading",
			State: "Fetching network config...",
		}
	}

	var err error
	var lsCfg *liteclient.GlobalConfig
	if netConfigPath != "" {
		log.Info().Msg("Fetching TON network config from disk...")
		lsCfg, err = liteclient.GetConfigFromFile(netConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ton config: %w", err)
		}
	} else {
		log.Info().Msg("Fetching TON network config...")
		lsCfg, err = liteclient.GetConfigFromUrl(context.Background(), "https://ton.org/global.config.json")
		if err != nil {
			log.Error().Err(err).Msg("Failed to download ton config; taking it from static cache")
			lsCfg = &liteclient.GlobalConfig{}
			if err = json.NewDecoder(bytes.NewBufferString(config.FallbackNetworkConfig)).Decode(lsCfg); err != nil {
				return nil, fmt.Errorf("failed to parse fallback ton config: %w", err)
			}
		}
	}

	return StartProxyWithConfig(addr, verbosity, res, blockHttp, versionAndDevice, lsCfg, tunnelConfigPath)
}

var ErrGenerated = errors.New("generated tunnel config; fill it with the desired route and restart")

func StartProxyWithConfig(addr string, verbosity int, res chan<- State, blockHttp bool, versionAndDevice string, lsCfg *liteclient.GlobalConfig, tunnelConfigPath string) (*Proxy, error) {
	report := func(s State) {
		if res != nil {
			res <- s
		}
	}

	var netMgr adnl.NetManager
	if tunnelConfigPath != "" {
		data, err := os.ReadFile(tunnelConfigPath)
		if err == nil && len(data) == 0 {
			err = os.Remove(tunnelConfigPath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to remove empty tunnel config")
				return nil, err
			}
			// to replace empty
			err = os.ErrNotExist
		}

		if err != nil {
			if os.IsNotExist(err) {
				if _, err = tunnelConfig.GenerateClientConfig(tunnelConfigPath); err != nil {
					log.Error().Err(err).Msg("Failed to generate tunnel config")
					return nil, err
				}
				log.Info().Msg("Generated tunnel config; fill it with the desired route and restart")
				return nil, ErrGenerated
			}
			log.Error().Err(err).Msg("Failed to load tunnel config")
			return nil, err
		}

		var tunCfg tunnelConfig.ClientConfig
		if err = json.Unmarshal(data, &tunCfg); err != nil {
			log.Error().Err(err).Msg("Failed to parse tunnel config")
			return nil, err
		}

		tun, port, ip, err := tunnel.PrepareTunnel(&tunCfg, lsCfg)
		if err != nil {
			log.Error().Err(err).Msg("Tunnel preparation failed")
			return nil, err
		}
		netMgr = adnl.NewMultiNetReader(tun)

		log.Info().Str("ip", ip.String()).Uint16("port", port).Msg("Using tunnel")
	} else {
		dl, err := adnl.DefaultListener(":")
		if err != nil {
			log.Error().Err(err).Msg("Failed to create default listener")
			return nil, err
		}
		netMgr = adnl.NewMultiNetReader(dl)
	}

	listenThreads := runtime.NumCPU()
	if listenThreads > 32 {
		listenThreads = 32
	}

	report(State{
		Type:  "loading",
		State: "Initializing DHT...",
	})

	log.Info().Msg("Initialising DHT client...")
	_, dhtAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 dht adnl key: %w", err)
	}

	gateway := adnl.NewGatewayWithNetManager(dhtAdnlKey, netMgr)
	err = gateway.StartClient()
	if err != nil {
		return nil, fmt.Errorf("failed to start adnl gateway: %w", err)
	}

	dhtClient, err := dht.NewClientFromConfig(gateway, lsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init DHT client: %w", err)
	}

	report(State{
		Type:  "loading",
		State: "Initializing DNS...",
	})

	log.Info().Msg("Initialising DNS resolver...")
	connPool, dnsClient, err := initDNSResolver(lsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init TON DNS resolver: %w", err)
	}

	report(State{
		Type:  "loading",
		State: "Initializing RLDP...",
	})

	log.Info().Msg("Initialising RLDP transport layer...")
	_, storageAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 storage adnl key: %w", err)
	}
	_, proxyAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 proxy adnl key: %w", err)
	}

	gateStorage := adnl.NewGatewayWithNetManager(storageAdnlKey, netMgr)
	if err = gateStorage.StartClient(listenThreads); err != nil {
		return nil, fmt.Errorf("failed to init adnl gateway: %w", err)
	}

	srv := storage.NewServer(dhtClient, gateStorage, storageAdnlKey, false)
	conn := storage.NewConnector(srv)

	store := transport.NewVirtualStorage()
	srv.SetStorage(store)

	gateProxy := adnl.NewGatewayWithNetManager(proxyAdnlKey, netMgr)
	if err = gateProxy.StartClient(listenThreads); err != nil {
		return nil, fmt.Errorf("failed to init adnl gateway for proxy: %w", err)
	}

	t := transport.NewTransport(gateProxy, dhtClient, dnsClient, conn, store)
	client = &http.Client{
		Transport: t,
	}

	report(State{
		Type:  "ready",
		State: "Ready",
	})

	log.Info().Str("address", addr).Msg("Starting proxy server")

	server := http.Server{Addr: addr, Handler: &proxy{blockHttp: blockHttp, version: versionAndDevice}}

	go func() {
		if err = server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			log.Error().Err(err).Msg("Failed to init proxy server")

			text := "Failed, check logs"
			if strings.Contains(err.Error(), "address already in use") {
				text = "Port is already in use"
			}

			report(State{
				Type:    "error",
				State:   text,
				Stopped: true,
			})
		}
	}()

	return &Proxy{
		httpServer:  &server,
		dht:         dhtClient,
		tr:          t,
		st:          srv,
		gateStorage: gateStorage,
		gateProxy:   gateProxy,
		connPool:    connPool,
	}, nil
}

func (p *Proxy) Stop() {
	p.httpServer.Close()

	p.connPool.Stop()
	p.st.Stop()
	p.dht.Close()
	p.gateProxy.Close()
	p.gateStorage.Close()
	p.tr.Stop()
}

func initDNSResolver(cfg *liteclient.GlobalConfig) (*liteclient.ConnectionPool, *dns.Client, error) {
	pool := liteclient.NewConnectionPool()

	// connect to testnet lite server
	err := pool.AddConnectionsFromConfig(context.Background(), cfg)
	if err != nil {
		return nil, nil, err
	}

	// initialize ton api lite connection wrapper
	api := ton.NewAPIClient(pool)

	var root *address.Address
	for i := 0; i < 5; i++ { // retry to not get liteserver not found block err
		// get root dns address from network config
		root, err = dns.RootContractAddr(api)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
	if err != nil {
		return nil, nil, err
	}

	return pool, dns.NewDNSClient(api, root), nil
}
