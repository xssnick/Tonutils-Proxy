package main

import "C"
import (
	"fmt"
	"github.com/xssnick/tonutils-proxy/proxy"
	"log"
)

func main() {}

//export StartProxy
func StartProxy(port C.ushort) *C.char {
	err := proxy.StartProxy("127.0.0.1:"+fmt.Sprint(uint16(port)), false, nil, false)
	if err != nil {
		log.Println("failed to start proxy:", err.Error())
		return C.CString("ERR")
	}
	return C.CString("OK")
}
