package main

import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-proxy/proxy"
	"log"
)

var GitCommit string

func main() {}

var ActiveProxy context.Context
var ProxyStopper context.CancelFunc

func init() {
	ActiveProxy, ProxyStopper = context.WithCancel(context.Background())
	ProxyStopper() // mark it not started
}

//export StartProxy
func StartProxy(port C.ushort) *C.char {
	return C.CString(startProxy(uint16(port), nil))
}

//export StartProxyWithConfig
func StartProxyWithConfig(port C.ushort, configTextJSON *C.char) *C.char {
	var cfg liteclient.GlobalConfig
	if err := json.Unmarshal([]byte(C.GoString(configTextJSON)), &cfg); err != nil {
		log.Println("failed to parse config:", err.Error())
		return C.CString("PARSE_CONFIG_ERR: " + err.Error())
	}

	return C.CString(startProxy(uint16(port), &cfg))
}

//export StopProxy
func StopProxy() *C.char {
	ProxyStopper()
	return C.CString("OK")
}

func startProxy(port uint16, cfg *liteclient.GlobalConfig) string {
	select {
	case <-ActiveProxy.Done():
	default:
		return "ALREADY_STARTED"
	}

	ActiveProxy, ProxyStopper = context.WithCancel(context.Background())

	var ch = make(chan proxy.State, 1)
	var err error
	go func() {
		if cfg != nil {
			err = proxy.RunProxyWithConfig(ActiveProxy, "127.0.0.1:"+fmt.Sprint(port), nil, nil, false, "LIB "+GitCommit, cfg, nil, nil)
		} else {
			err = proxy.RunProxy(ActiveProxy, "127.0.0.1:"+fmt.Sprint(port), nil, ch, "LIB "+GitCommit, false, "", nil, nil)
		}
		if err != nil {
			log.Println("failed to start proxy:", err.Error())
			ch <- proxy.State{Type: "error", State: err.Error(), Stopped: true}
		}
	}()

	var res = make(chan string, 1)
	go func() {
		for {
			select {
			case <-ActiveProxy.Done():
				return
			case state := <-ch:
				var msg string
				if state.Stopped {
					ProxyStopper()
					msg = "ERR: " + state.State
				} else if state.Type == "ready" {
					msg = "OK"
				}

				select {
				case res <- msg:
				default:
				}
			}
		}
	}()

	return <-res
}
