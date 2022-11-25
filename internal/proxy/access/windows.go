//go:build windows

package access

import "C"
import (
	"errors"
)

func enableProxy(addr, port string) error {
	return errors.New("automatic system configuration is not yet implemented for your platform")
}

func disableProxy() error {
	return errors.New("automatic system configuration is not yet implemented for your platform")
}
