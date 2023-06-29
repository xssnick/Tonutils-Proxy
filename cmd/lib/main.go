package main

import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-proxy/proxy"
	"log"
)

func main() {}

var ActiveProxy *proxy.Proxy

//export StartProxy
func StartProxy(port C.ushort) *C.char {
	if ActiveProxy != nil {
		return C.CString("ALREADY_STARTED")
	}

	p, err := proxy.StartProxy("127.0.0.1:"+fmt.Sprint(uint16(port)), false, nil, false)
	if err != nil {
		log.Println("failed to start proxy:", err.Error())
		return C.CString("ERR: " + err.Error())
	}
	ActiveProxy = p
	return C.CString("OK")
}

//export StartProxyWithConfig
func StartProxyWithConfig(port C.ushort, configTextJSON *C.char) *C.char {
	if ActiveProxy != nil {
		return C.CString("ALREADY_STARTED")
	}

	var cfg liteclient.GlobalConfig
	if err := json.Unmarshal([]byte(C.GoString(configTextJSON)), &cfg); err != nil {
		log.Println("failed to parse config:", err.Error())
		return C.CString("PARSE_CONFIG_ERR: " + err.Error())
	}

	p, err := proxy.StartProxyWithConfig("127.0.0.1:"+fmt.Sprint(uint16(port)), false, nil, false, &cfg)
	if err != nil {
		log.Println("failed to start proxy:", err.Error())
		return C.CString("ERR: " + err.Error())
	}
	ActiveProxy = p
	return C.CString("OK")
}

//export StopProxy
func StopProxy() *C.char {
	if ActiveProxy != nil {
		ActiveProxy.Stop()
	}
	return C.CString("OK")
}
