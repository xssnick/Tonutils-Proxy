//go:build darwin

package access

import "C"
import (
	"sync"
	"unsafe"
)

/*
#cgo CFLAGS: -x objective-c -fmodules
#cgo LDFLAGS: -framework Foundation

#import <Foundation/NSArray.h>
#import <Foundation/Foundation.h>
#import <SystemConfiguration/SCPreferences.h>
#import <SystemConfiguration/SCNetworkConfiguration.h>

#include <sys/syslimits.h>
#include <sys/stat.h>
#include <mach-o/dyld.h>

char* proxyHost;
char* proxyPort;

enum RET_ERRORS {
  RET_NO_ERROR = 0,
  INVALID_FORMAT = 1,
  NO_PERMISSION = 2,
  SYSCALL_FAILED = 3,
  NO_MEMORY = 4
};

Boolean toggleAction(SCNetworkProtocolRef proxyProtocolRef, NSDictionary* oldPreferences, bool turnOn) {
  NSString* nsProxyHost = [[NSString alloc] initWithCString: proxyHost encoding:NSUTF8StringEncoding];
  NSNumber* nsProxyPort = [[NSNumber alloc] initWithLong: [[[NSString alloc] initWithCString: proxyPort encoding:NSUTF8StringEncoding] integerValue]];
  NSString* nsOldProxyHost;
  NSNumber* nsOldProxyPort;
  NSMutableDictionary *newPreferences = [NSMutableDictionary dictionaryWithDictionary: oldPreferences];
  Boolean success;

  if (turnOn) {
    [newPreferences setValue: nsProxyHost forKey:(NSString*)kSCPropNetProxiesHTTPProxy];
    [newPreferences setValue: nsProxyPort forKey:(NSString*)kSCPropNetProxiesHTTPPort];
    [newPreferences setValue:[NSNumber numberWithInt:1] forKey:(NSString*)kSCPropNetProxiesHTTPEnable];
  } else {
    nsOldProxyHost = [newPreferences valueForKey:(NSString*)kSCPropNetProxiesHTTPProxy];
    nsOldProxyPort = [newPreferences valueForKey:(NSString*)kSCPropNetProxiesHTTPPort];
    if ([nsProxyHost isEqualToString:nsOldProxyHost] && [nsProxyPort intValue] == [nsOldProxyPort intValue]) {
      [newPreferences setValue:[NSNumber numberWithInt:0] forKey:(NSString*)kSCPropNetProxiesHTTPEnable];
      [newPreferences setValue: @"" forKey:(NSString*)kSCPropNetProxiesHTTPProxy];
      [newPreferences setValue: @"" forKey:(NSString*)kSCPropNetProxiesHTTPPort];
    }
  }

  success = SCNetworkProtocolSetConfiguration(proxyProtocolRef, (__bridge CFDictionaryRef)newPreferences);
  if(!success) {
    NSLog(@"Failed to set Protocol Configuration");
  }
  return success;
}

int toggle(bool turnOn, AuthorizationRef auth) {
  int ret = RET_NO_ERROR;
  Boolean success;

  SCNetworkSetRef networkSetRef;
  CFArrayRef networkServicesArrayRef;
  SCNetworkServiceRef networkServiceRef;
  SCNetworkProtocolRef proxyProtocolRef;
  NSDictionary *oldPreferences;

  // Get System Preferences Lock
  SCPreferencesRef prefsRef = SCPreferencesCreateWithAuthorization(NULL, CFSTR("org.tonutils.proxy"), NULL, auth);

  if(prefsRef==NULL) {
    NSLog(@"Fail to obtain Preferences Ref");
    ret = NO_PERMISSION;
    goto freePrefsRef;
  }

  success = SCPreferencesLock(prefsRef, true);
  if (!success) {
    NSLog(@"Fail to obtain PreferencesLock");
    ret = NO_PERMISSION;
    goto freePrefsRef;
  }

  // Get available network services
  networkSetRef = SCNetworkSetCopyCurrent(prefsRef);
  if(networkSetRef == NULL) {
    NSLog(@"Fail to get available network services");
    ret = SYSCALL_FAILED;
    goto freeNetworkSetRef;
  }

  //Look up interface entry
  networkServicesArrayRef = SCNetworkSetCopyServices(networkSetRef);
  networkServiceRef = NULL;
  for (long i = 0; i < CFArrayGetCount(networkServicesArrayRef); i++) {
    networkServiceRef = CFArrayGetValueAtIndex(networkServicesArrayRef, i);

    // Get proxy protocol
    proxyProtocolRef = SCNetworkServiceCopyProtocol(networkServiceRef, kSCNetworkProtocolTypeProxies);
    if(proxyProtocolRef == NULL) {
      NSLog(@"Couldn't acquire copy of proxyProtocol");
      ret = SYSCALL_FAILED;
      goto freeProxyProtocolRef;
    }

    oldPreferences = (__bridge NSDictionary*)SCNetworkProtocolGetConfiguration(proxyProtocolRef);
    if (!toggleAction(proxyProtocolRef, oldPreferences, turnOn)) {
      ret = SYSCALL_FAILED;
    }

freeProxyProtocolRef:
    CFRelease(proxyProtocolRef);
  }

	success = SCPreferencesCommitChanges(prefsRef);
	if(!success) {
	  NSLog(@"Failed to Commit Changes");
	  ret = SYSCALL_FAILED;
	  goto freeNetworkServicesArrayRef;
	}

	success = SCPreferencesApplyChanges(prefsRef);
	if(!success) {
	  NSLog(@"Failed to Apply Changes");
	  ret = SYSCALL_FAILED;
	  goto freeNetworkServicesArrayRef;
	}


  //Free Resources
freeNetworkServicesArrayRef:
  CFRelease(networkServicesArrayRef);
freeNetworkSetRef:
  CFRelease(networkSetRef);
freePrefsRef:
  SCPreferencesUnlock(prefsRef);
  CFRelease(prefsRef);

  return ret;
}

AuthorizationRef auth() {
    AuthorizationRef a;
    OSStatus status = AuthorizationCreate(
        NULL,
        kAuthorizationEmptyEnvironment,
        kAuthorizationFlagInteractionAllowed | kAuthorizationFlagPreAuthorize | kAuthorizationFlagExtendRights,
        &a
    );
    if (status != errAuthorizationSuccess) {
        return NULL;
    }
    return a;
}

int setProxy(char* host, char* port, bool enabled, AuthorizationRef auth) {
	proxyHost = host;
	proxyPort = port;

    return toggle(enabled, auth);
}
*/
import "C"

var authP C.AuthorizationRef
var once sync.Once

var setAddr, setPort string

func auth() {
	once.Do(func() {
		authP = C.auth()
	})
}

func enableProxy(addr, port string) error {
	auth()

	setAddr = addr
	setPort = port

	cAddr := C.CString(addr)
	defer C.free(unsafe.Pointer(cAddr))

	cPort := C.CString(port)
	defer C.free(unsafe.Pointer(cPort))

	enable := C.bool(true)
	C.setProxy(cAddr, cPort, enable, authP)

	return nil
}

func disableProxy() error {
	if authP == nil { // proxy was not set
		return nil
	}

	cAddr := C.CString(setAddr)
	defer C.free(unsafe.Pointer(cAddr))

	cPort := C.CString(setPort)
	defer C.free(unsafe.Pointer(cPort))

	enable := C.bool(false)
	C.setProxy(cAddr, cPort, enable, authP)

	return nil
}
