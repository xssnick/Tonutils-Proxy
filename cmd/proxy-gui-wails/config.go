package main

import (
	"crypto/ed25519"
	"encoding/json"
	tunnelConfig "github.com/ton-blockchain/adnl-tunnel/config"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Config struct {
	Version         uint
	ProxyListenAddr string
	ADNLKey         []byte

	NetworkConfigPath             string
	CustomTunnelNetworkConfigPath string
	TunnelConfig                  *tunnelConfig.ClientConfig

	mx sync.Mutex
}

func LoadConfig(dir string) (*Config, error) {
	var cfg *Config
	path := dir + "/config.json"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, err
		}

		cfg = &Config{
			Version:         1,
			ProxyListenAddr: "127.0.0.1:8080",
			ADNLKey:         priv.Seed(),
		}

		cfg.TunnelConfig, err = tunnelConfig.GenerateClientConfig()
		if err != nil {
			return nil, err
		}
		cfg.TunnelConfig.Payments.DBPath = dir + "/payments-db"

		if err = cfg.SaveConfig(dir); err != nil {
			return nil, err
		}

		return cfg, nil
	} else if err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(data, &cfg)
		if err != nil {
			return nil, err
		}

		if cfg.ADNLKey == nil {
			_, priv, err := ed25519.GenerateKey(nil)
			if err != nil {
				return nil, err
			}
			cfg.ADNLKey = priv.Seed()
			_ = cfg.SaveConfig(dir)
		}
	}

	if cfg.Version < 1 {
		var err error

		cfg.Version = 1
		cfg.TunnelConfig, err = tunnelConfig.GenerateClientConfig()
		if err != nil {
			return nil, err
		}
		cfg.TunnelConfig.Payments.DBPath = dir + "/payments-db"

		err = cfg.SaveConfig(dir)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func (cfg *Config) SaveConfig(dir string) error {
	cfg.mx.Lock()
	defer cfg.mx.Unlock()

	path := dir + "/config.json"

	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(path, data, 0766)
	if err != nil {
		return err
	}
	return nil
}

var CustomRoot = ""

func PrepareRootPath() (string, error) {
	if CustomRoot != "" {
		return CustomRoot, nil
	}

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		path := home + "/Library/Application Support/org.tonutils.proxy"
		_, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(path, 0766)
			}
			if err != nil {
				return "", err
			}
		}
		return path, nil
	case "windows":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		path := home + "\\AppData\\Roaming\\Tonutils Proxy.exe"
		_, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(path, 0766)
			}
			if err != nil {
				return "", err
			}
		}
		return path, nil
	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex), nil
}
