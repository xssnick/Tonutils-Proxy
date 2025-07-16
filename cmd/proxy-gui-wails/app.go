package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/ton-blockchain/adnl-tunnel/config"
	"github.com/ton-blockchain/adnl-tunnel/tunnel"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xssnick/ton-payment-network/tonpayments/wallet"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-proxy/proxy"
	"github.com/xssnick/tonutils-proxy/proxy/access"
	"math/big"
	"os"
	"os/exec"
	rt "runtime"
	"sync"
)

// App struct
type App struct {
	ctx context.Context

	proxyStopCtx context.Context
	proxyStop    context.CancelFunc
	statusUpd    chan proxy.State
	rootPath     string

	skipTunnel bool

	tunnelGracefulStopCtx context.Context
	tunnelGracefulStop    context.CancelFunc

	cfg *Config
}

// NewApp creates a new App application struct
func NewApp() (*App, error) {
	cfgDir, err := PrepareRootPath()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare root path: %w", err)
	}

	cfg, err := LoadConfig(cfgDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	proxyStopCtx, proxyStop := context.WithCancel(context.Background())
	proxyStop()

	tunnelGracefulStopCtx, tunnelGracefulStop := context.WithCancel(context.Background())
	tunnelGracefulStop()

	a := &App{
		rootPath:              cfgDir,
		cfg:                   cfg,
		proxyStopCtx:          proxyStopCtx,
		proxyStop:             proxyStop,
		tunnelGracefulStopCtx: tunnelGracefulStopCtx,
		tunnelGracefulStop:    tunnelGracefulStop,
	}

	proxy.OnAskAccept = func(to, from []*tunnel.SectionInfo) int {
		var priceIn, priceOut = big.NewInt(0), big.NewInt(0)
		var sect []SectionInfo
		for i, n := range append(to, from...) {
			sect = append(sect, SectionInfo{
				Name:  base64.StdEncoding.EncodeToString(n.Keys.ReceiverPubKey)[:8],
				Outer: i == len(to)-1,
			})

			if n.PaymentInfo != nil {
				if n.PaymentInfo.ExtraCurrencyID != 0 || n.PaymentInfo.JettonMaster != nil {
					a.ShowWarnMsg("Route has node with payment in currency other than TON, it is not yet supported in Torrent, rerouting")
					return tunnel.AcceptorDecisionCancel
				}

				// consider 1 packet = 512 bytes, actually more, but this is avg payload
				var packetsPerMB int64 = 2048

				amt := new(big.Int).SetUint64(n.PaymentInfo.PricePerPacket)
				amt.Mul(amt, big.NewInt(packetsPerMB))

				vcFee := big.NewInt(0)
				for _, section := range n.PaymentInfo.PaymentTunnel {
					vcFee.Add(vcFee, section.MinFee)
				}

				packetsPerChannel := tunnel.ChannelCapacityForNumPayments * tunnel.ChannelPacketsToPrepay
				// channel fee per 1 mb
				feeDiv := new(big.Float).Quo(new(big.Float).SetInt64(packetsPerMB), new(big.Float).SetInt64(packetsPerChannel))

				feePer1MB, _ := feeDiv.Mul(new(big.Float).SetInt(vcFee), feeDiv).Int(vcFee)
				amt.Add(amt, feePer1MB)

				if i < len(to)-1 {
					priceOut.Add(priceOut, amt)
				} else if i == len(to)-1 {
					priceOut.Add(priceOut, amt)
					priceIn.Add(priceOut, amt)
				} else {
					priceIn.Add(priceOut, amt)
				}
			}
		}

		runtime.EventsEmit(a.ctx, "tunnel_check", sect, tlb.FromNanoTON(priceIn).String(), tlb.FromNanoTON(priceOut).String())

		ch := make(chan int, 1)
		runtime.EventsOn(a.ctx, "tunnel_check_result", func(optionalData ...interface{}) {
			runtime.EventsOff(a.ctx, "tunnel_check_result")
			if len(optionalData) == 0 {
				// cancel tunnel, start without it
				a.skipTunnel = true
				ch <- tunnel.AcceptorDecisionCancel
				return
			}

			if optionalData[0].(bool) {
				ch <- tunnel.AcceptorDecisionAccept
			} else {
				ch <- tunnel.AcceptorDecisionReject
				if len(optionalData) > 1 {
					a.cfg.TunnelConfig.TunnelSectionsNum = uint(optionalData[1].(float64))
					a.SaveTunnelConfig(a.cfg.TunnelConfig.TunnelSectionsNum, true, a.cfg.TunnelConfig.NodesPoolConfigPath)
				}
			}
		})

		select {
		case <-a.proxyStopCtx.Done():
			return tunnel.AcceptorDecisionCancel
		case v := <-ch:
			return v
		}
	}

	proxy.OnAskReroute = func() bool {
		runtime.EventsEmit(a.ctx, "tunnel_reinit_ask")

		ch := make(chan bool, 1)
		runtime.EventsOn(a.ctx, "tunnel_reinit_ask_result", func(optionalData ...interface{}) {
			runtime.EventsOff(a.ctx, "tunnel_reinit_ask_result")
			ch <- optionalData[0].(bool)
		})

		select {
		case <-a.proxyStopCtx.Done():
			return false
		case v := <-ch:
			return v
		}
	}

	proxy.OnPaidUpdate = func(paid tlb.Coins) {
		runtime.EventsEmit(a.ctx, "tunnel_paid", paid.String())
	}

	proxy.OnTunnel = func(addr string) {
		runtime.EventsEmit(a.ctx, "tunnel_updated", addr)
	}

	proxy.OnTunnelStopped = func() {
		a.tunnelGracefulStop()
	}

	return a, nil
}

type SectionInfo struct {
	Name  string
	Outer bool
}

func (a *App) DummySec() []SectionInfo {
	return []SectionInfo{}
}

func (a *App) GetProxyAddr() string {
	return a.cfg.ProxyListenAddr
}

func (a *App) GetTunnelNodesConfigPath() string {
	if a.cfg.TunnelConfig != nil {
		return a.cfg.TunnelConfig.NodesPoolConfigPath
	}
	return ""
}

func (a *App) GetConfig() *Config {
	return a.cfg
}

func (a *App) GetPaymentNetworkWalletAddr() string {
	w, err := wallet.InitWallet(ton.NewAPIClient(liteclient.NewOfflineClient()), ed25519.NewKeyFromSeed(a.cfg.TunnelConfig.Payments.WalletPrivateKey))
	if err != nil {
		log.Error().Err(err).Msg("init wallet error")
		return "{ERROR}"
	}
	return w.WalletAddress().String()
}

func (a *App) SaveTunnelConfig(num uint, payments bool, poolPath string) string {
	a.cfg.TunnelConfig.TunnelSectionsNum = num
	a.cfg.TunnelConfig.PaymentsEnabled = payments
	a.cfg.TunnelConfig.NodesPoolConfigPath = poolPath

	err := a.cfg.SaveConfig(a.rootPath)
	if err != nil {
		log.Error().Err(err).Msg("save config error")
		return err.Error()
	}

	runtime.EventsEmit(a.ctx, "config_saved")
	return ""
}

func (a *App) ShowWarnMsg(text string) {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.WarningDialog,
		Title:         "Warning",
		Message:       text,
		DefaultButton: "OK",
	})
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.statusUpd = make(chan proxy.State, 1)

	go func() {
		for {
			state := <-a.statusUpd
			runtime.EventsEmit(a.ctx, "statusUpdate", state.Type, state.State)

			if state.Stopped {
				a.proxyStop()
			} else if state.Type == "ready" {
				if err := access.SetProxy(a.cfg.ProxyListenAddr); err != nil {
					println(err.Error())
				} else {
					openOnce.Do(func() {
						openbrowser("http://foundation.ton/")
					})
				}
			}
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	select {
	case <-a.proxyStopCtx.Done():
	default:
		a.proxyStop()
		log.Info().Msg("Clearing proxy")
		_ = access.ClearProxy()
	}

	log.Info().Msg("waiting for graceful stop")
	<-a.tunnelGracefulStopCtx.Done()
	log.Info().Msg("gracefully stopped")
}

var openOnce sync.Once

func (a *App) ResetTunnelConfig() {
	a.SaveTunnelConfig(a.cfg.TunnelConfig.TunnelSectionsNum, true, "")
	runtime.EventsEmit(a.ctx, "tunnel_pool_added", "", 0)
}

func (a *App) AddTunnel() {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: "",
		DefaultFilename:  "nodes-pool.json",
		Title:            "Open Tunnel Nodes Pool",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "nodes-pool.json",
				Pattern:     "*.json",
			},
		},
		ShowHiddenFiles:            false,
		CanCreateDirectories:       false,
		ResolvesAliases:            false,
		TreatPackagesAsDirectories: false,
	})
	if err != nil {
		println(err.Error())
	}

	if path != "" {
		res := a.parseTunnelConfig(a.cfg.TunnelConfig.NodesPoolConfigPath)
		if res != nil {
			a.SaveTunnelConfig(a.cfg.TunnelConfig.TunnelSectionsNum, true, path)
			runtime.EventsEmit(a.ctx, "tunnel_pool_added", path, res.Max)
			return
		}
	}

	a.SaveTunnelConfig(a.cfg.TunnelConfig.TunnelSectionsNum, true, "")
	runtime.EventsEmit(a.ctx, "tunnel_pool_added", "", 0)
	return
}

func (a *App) GetMaxTunnelNodes() int {
	res := a.parseTunnelConfig(a.cfg.TunnelConfig.NodesPoolConfigPath)
	if res == nil {
		return 0
	}

	return res.Max
}

type TunnelConfigInfo struct {
	Max     int
	MaxFree int
	Path    string
}

func (a *App) parseTunnelConfig(path string) *TunnelConfigInfo {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:          runtime.ErrorDialog,
				Title:         "Failed to read tunnel config",
				Message:       err.Error(),
				DefaultButton: "Ok",
			})
			return nil
		}

		var sharedCfg config.SharedConfig
		if err = json.Unmarshal(data, &sharedCfg); err != nil {
			_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:          runtime.ErrorDialog,
				Title:         "Failed to parse tunnel config",
				Message:       err.Error(),
				DefaultButton: "Ok",
			})
			return nil
		}

		if len(sharedCfg.NodesPool) == 0 {
			_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "Failed to parse nodes pool config",
				Message: "Unable to load tunnel node settings\nPlease check that the configuration file is correct and contains at least one node",
			})
			return nil
		}

		maxFree := 0
		for _, node := range sharedCfg.NodesPool {
			if node.Payment == nil {
				maxFree++
			}
		}

		return &TunnelConfigInfo{Path: path, Max: len(sharedCfg.NodesPool), MaxFree: maxFree}
	}

	return &TunnelConfigInfo{Path: ""}
}

func (a *App) StartProxy() {
	select {
	case <-a.proxyStopCtx.Done():
	default:
		return
	}

	a.proxyStopCtx, a.proxyStop = context.WithCancel(a.ctx)

	go func() {
		defer a.StopProxy()

		var err error
		var customTunNetCfg *liteclient.GlobalConfig
		if a.cfg.CustomTunnelNetworkConfigPath != "" {
			customTunNetCfg, err = liteclient.GetConfigFromFile(a.cfg.CustomTunnelNetworkConfigPath)
			if err != nil {
				a.ShowWarnMsg(err.Error())
				log.Fatal().Err(err).Msg("failed to load custom net config for tun")
			}
		}

		tun := a.cfg.TunnelConfig

		if tun != nil && tun.NodesPoolConfigPath != "" {
			a.tunnelGracefulStopCtx, a.tunnelGracefulStop = context.WithCancel(context.Background())
		}

	retry:
		err = proxy.RunProxy(a.proxyStopCtx, a.cfg.ProxyListenAddr, a.cfg.ADNLKey, a.statusUpd, "GUI 1.7", false, "", tun, customTunNetCfg)
		if err != nil {
			if a.skipTunnel {
				a.skipTunnel = false
				tun = nil // retry without tunnel
				goto retry
			}

			_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "Startup",
				Message: err.Error(),
			})
		}
	}()
}

func (a *App) StopProxy() {
	runtime.EventsEmit(a.ctx, "statusUpdate", "loading", "stopping")

	select {
	case <-a.proxyStopCtx.Done():
	default:
		a.proxyStop()
		_ = access.ClearProxy()
	}

	runtime.EventsEmit(a.ctx, "statusUpdate", "stopped", "stopped")
}

func openbrowser(url string) {
	var err error

	switch rt.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Error().Err(err).Msg("cannot open browser")
	}
}
