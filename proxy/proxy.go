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
	adnlAddress "github.com/xssnick/tonutils-go/adnl/address"
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
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

func RunProxy(closerCtx context.Context, addr string, adnlKey ed25519.PrivateKey, res chan<- State, versionAndDevice string, blockHttp bool, netConfigPath string, tunCfg *tunnelConfig.ClientConfig, customTunNetCfg *liteclient.GlobalConfig) error {
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
			return fmt.Errorf("failed to parse ton config: %w", err)
		}
	} else {
		log.Info().Msg("Fetching TON network config...")
		lsCfg, err = liteclient.GetConfigFromUrl(context.Background(), "https://ton-blockchain.github.io/global.config.json")
		if err != nil {
			log.Error().Err(err).Msg("Failed to download ton config; taking it from static cache")
			lsCfg = &liteclient.GlobalConfig{}
			if err = json.NewDecoder(bytes.NewBufferString(config.FallbackNetworkConfig)).Decode(lsCfg); err != nil {
				return fmt.Errorf("failed to parse fallback ton config: %w", err)
			}
		}
	}

	return RunProxyWithConfig(closerCtx, addr, adnlKey, res, blockHttp, versionAndDevice, lsCfg, tunCfg, customTunNetCfg)
}

var OnTunnel = func(addr string) {}
var OnPaidUpdate = func(paid tlb.Coins) {}

var OnAskAccept = func(to, from []*tunnel.SectionInfo) int {
	return tunnel.AcceptorDecisionAccept
}
var OnAskReroute = func() bool { return false }

var OnTunnelStopped = func() {}

func RunProxyWithConfig(closerCtx context.Context, addr string, adnlKey ed25519.PrivateKey, res chan<- State, blockHttp bool, versionAndDevice string, lsCfg *liteclient.GlobalConfig, tunCfg *tunnelConfig.ClientConfig, customTunNetCfg *liteclient.GlobalConfig) error {
	report := func(s State) {
		if res != nil {
			res <- s
		}
	}

	var err error
	if len(adnlKey) == 0 {
		_, adnlKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			return fmt.Errorf("failed to generate ed25519 adnl key: %w", err)
		}
	}

	ctx, closer := context.WithCancel(closerCtx)
	defer closer()

	report(State{
		Type:  "loading",
		State: "Initializing DNS...",
	})

	log.Info().Msg("Initializing DNS resolver...")
	connPool, dnsClient, err := initDNSResolver(lsCfg)
	if err != nil {
		return fmt.Errorf("failed to init TON DNS resolver: %w", err)
	}
	defer connPool.Stop()

	var gate *adnl.Gateway
	var netMgr adnl.NetManager
	if tunCfg != nil && tunCfg.NodesPoolConfigPath != "" {
		report(State{
			Type:  "loading",
			State: "Preparing ADNL tunnel...",
		})

		data, err := os.ReadFile(tunCfg.NodesPoolConfigPath)
		if err != nil {
			return fmt.Errorf("failed to load tunnel nodes pool config: %w", err)
		}

		var tunNodesCfg tunnelConfig.SharedConfig
		if err = json.Unmarshal(data, &tunNodesCfg); err != nil {
			return fmt.Errorf("failed to parse tunnel nodes pool config: %w", err)
		}

		if customTunNetCfg == nil {
			customTunNetCfg = lsCfg
		}

		tunnel.ChannelPacketsToPrepay = 30000
		tunnel.ChannelCapacityForNumPayments = 50

		tunnel.AskReroute = OnAskReroute
		tunnel.Acceptor = OnAskAccept
		events := make(chan any, 1)
		go tunnel.RunTunnel(ctx, tunCfg, &tunNodesCfg, customTunNetCfg, log.Logger, events)

		initUpd := make(chan any, 1)
		inited := false
		go func() {
			atm := &tunnel.AtomicSwitchableRegularTunnel{}
			for event := range events {
				switch e := event.(type) {
				case tunnel.StoppedEvent:
					OnTunnelStopped()
					return
				case tunnel.MsgEvent:
					if !inited {
						report(State{
							Type:  "loading",
							State: e.Msg,
						})
					}
				case tunnel.UpdatedEvent:
					log.Info().Msg("tunnel updated")

					e.Tunnel.SetOutAddressChangedHandler(func(addr *net.UDPAddr) {
						gate.SetAddressList([]*adnlAddress.UDP{
							{
								IP:   addr.IP,
								Port: int32(addr.Port),
							},
						})
						OnTunnel(addr.String())
					})
					OnTunnel(fmt.Sprintf("%s:%d", e.ExtIP.String(), e.ExtPort))

					go func() {
						for {
							select {
							case <-e.Tunnel.AliveCtx().Done():
								return
							case <-time.After(5 * time.Second):
								OnPaidUpdate(e.Tunnel.CalcPaidAmount()["TON"])
							}
						}
					}()

					atm.SwitchTo(e.Tunnel)
					if !inited {
						inited = true
						netMgr = adnl.NewMultiNetReader(atm)
						gate = adnl.NewGatewayWithNetManager(adnlKey, netMgr)

						select {
						case initUpd <- e:
						default:
						}
					} else {
						gate.SetAddressList([]*adnlAddress.UDP{
							{
								IP:   e.ExtIP,
								Port: int32(e.ExtPort),
							},
						})

						log.Info().Msg("connection switched to new tunnel")
					}
				case tunnel.ConfigurationErrorEvent:
					report(State{
						Type:  "loading",
						State: "Tunnel configuration error, will retry...",
					})
					log.Err(e.Err).Msg("tunnel configuration error, will retry...")
				case error:
					select {
					case initUpd <- e:
					default:
					}
				}
			}
		}()

		switch x := (<-initUpd).(type) {
		case tunnel.UpdatedEvent:
			log.Info().
				Str("ip", x.ExtIP.String()).
				Uint16("port", x.ExtPort).
				Msg("using tunnel")
		case error:
			return fmt.Errorf("tunnel preparation failed: %w", x)
		}
	} else {
		dl, err := adnl.DefaultListener(":")
		if err != nil {
			log.Error().Err(err).Msg("Failed to create default listener")
			return err
		}
		netMgr = adnl.NewMultiNetReader(dl)
		gate = adnl.NewGatewayWithNetManager(adnlKey, netMgr)
	}
	defer gate.Close()
	defer netMgr.Close()

	listenThreads := runtime.NumCPU()
	if listenThreads > 32 {
		listenThreads = 32
	}

	report(State{
		Type:  "loading",
		State: "Initializing DHT...",
	})

	log.Info().Msg("Initializing DHT client...")
	_, dhtAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate ed25519 dht adnl key: %w", err)
	}

	gateway := adnl.NewGatewayWithNetManager(dhtAdnlKey, netMgr)
	err = gateway.StartClient()
	if err != nil {
		return fmt.Errorf("failed to start adnl gateway: %w", err)
	}
	defer gateway.Close()

	dhtClient, err := dht.NewClientFromConfig(gateway, lsCfg)
	if err != nil {
		return fmt.Errorf("failed to init DHT client: %w", err)
	}
	defer dhtClient.Close()

	report(State{
		Type:  "loading",
		State: "Initializing RLDP...",
	})

	log.Info().Msg("Initializing RLDP transport layer...")
	_, storageAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate ed25519 storage adnl key: %w", err)
	}
	_, proxyAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate ed25519 proxy adnl key: %w", err)
	}

	gateStorage := adnl.NewGatewayWithNetManager(storageAdnlKey, netMgr)
	if err = gateStorage.StartClient(listenThreads); err != nil {
		return fmt.Errorf("failed to init adnl gateway: %w", err)
	}
	defer gateStorage.Close()

	srv := storage.NewServer(dhtClient, gateStorage, storageAdnlKey, false, 1)
	conn := storage.NewConnector(srv)

	store := transport.NewVirtualStorage()
	srv.SetStorage(store)

	defer srv.Stop()

	gateProxy := adnl.NewGatewayWithNetManager(proxyAdnlKey, netMgr)
	if err = gateProxy.StartClient(listenThreads); err != nil {
		return fmt.Errorf("failed to init adnl gateway for proxy: %w", err)
	}
	defer gateProxy.Close()

	report(State{
		Type:  "loading",
		State: "Starting HTTP server...",
	})

	t := transport.NewTransport(gateProxy, dhtClient, dnsClient, conn, store)
	client = &http.Client{
		Transport: t,
	}
	defer t.Stop()

	log.Info().Str("address", addr).Msg("Starting proxy server")

	server := http.Server{Addr: addr, Handler: &proxy{blockHttp: blockHttp, version: versionAndDevice}}

	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()

	failed := false
	go func() {
		// wait for server start
		time.Sleep(1 * time.Second)
		if failed {
			return
		}

		report(State{
			Type:  "ready",
			State: "Ready",
		})
	}()

	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}

	if err != nil {
		failed = true
		if strings.Contains(err.Error(), "address already in use") {
			err = fmt.Errorf("cannot start server, port %s is already in use by another application", addr)
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

	return err
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
		root, err = dns.GetRootContractAddr(context.Background(), api)
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
