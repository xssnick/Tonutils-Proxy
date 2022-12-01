//go:build windows

package access

import (
	"errors"
	"unsafe"
)

/*
#include <Windows.h>
#include <string.h>

enum RET_ERRORS {
	RET_NO_ERROR = 0,
	MISSING_KEY = 1,
	SET_ENABLE_PROXY_ERROR = 2,
	SET_HOSTANDPORT_PROXY_ERROR = 3
};

HKEY hKey;

int setPoxy(char* host) {
	DWORD proxyEnable = 0x00000001;

	if (RegOpenKeyEx(HKEY_CURRENT_USER, TEXT("SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Internet Settings"), 0, KEY_ALL_ACCESS, &hKey) != ERROR_SUCCESS)
	{
		return MISSING_KEY;
	}

	if (RegSetValueEx(hKey, TEXT("ProxyEnable"), 0, REG_DWORD, (const BYTE*)&proxyEnable, sizeof(proxyEnable)) != ERROR_SUCCESS)
	{
		return SET_ENABLE_PROXY_ERROR;
	}

	if (RegSetValueEx(hKey, TEXT("ProxyServer"), 0, REG_SZ, host, strlen(host)) != ERROR_SUCCESS)
	{
		return SET_HOSTANDPORT_PROXY_ERROR;
	}

		RegCloseKey(hKey);
}

int disPoxy() {
	DWORD proxyDisable = 0x00000000;

	if (RegOpenKeyEx(HKEY_CURRENT_USER, TEXT("SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Internet Settings"), 0, KEY_ALL_ACCESS, &hKey) != ERROR_SUCCESS)
	{
		return MISSING_KEY;
	}

	if (RegSetValueEx(hKey, TEXT("ProxyEnable"), 0, REG_DWORD, (const BYTE*)&proxyDisable, sizeof(proxyDisable)) != ERROR_SUCCESS)
	{
		return SET_ENABLE_PROXY_ERROR;
	}

		RegCloseKey(hKey);
}
*/
import "C"

func enableProxy(addr, port string) error {
	host := addr + ":" + port
	cHost := C.CString(host)
	defer C.free(unsafe.Pointer(cHost))

	res := C.setPoxy(cHost)
	resGo := int(res)
	switch resGo {
	case 1:
		return errors.New("can't set proxy, err: missing key")
	case 2:
		return errors.New("can't set proxy, err: failed enable proxy")
	case 3:
		return errors.New("can't set proxy, err: failed set host and port")
	}
	return nil
}

func disableProxy() error {
	res := C.disPoxy()
	resGo := int(res)
	switch resGo {
	case 1:
		return errors.New("can't set proxy, err: missing key")
	case 2:
		return errors.New("can't set proxy, err: failed enable proxy")
	}
	return nil
}
