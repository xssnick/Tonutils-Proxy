package access

import "strings"

func SetProxy(addr string) error {
	s := strings.Split(addr, ":")
	return enableProxy(s[0], s[1])
}

func ClearProxy() error {
	return disableProxy()
}
