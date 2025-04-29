ver := $(shell git log -1 --pretty=format:%h)

build-all-cli:
	mkdir -p build/cli
	echo "Building MAC CLI ARM"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-darwin-arm64 cmd/proxy-cli/main.go
	echo "Building MAC CLI AMD"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-darwin-amd64 cmd/proxy-cli/main.go
	echo "Building LINUX CLI AMD"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-linux-amd64 cmd/proxy-cli/main.go
	echo "Building WINDOWS CLI AMD"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-windows-amd64.exe cmd/proxy-cli/main.go

sdk=iphoneos
arch=arm64
sdk_path=$(shell xcrun --sdk ${sdk} --show-sdk-path)
clang_path=$(shell xcrun --sdk ${sdk} --find clang)
goRoot=$(shell go env GOROOT)

build-ios-lib:
	SDK=$(sdk) CGO_ENABLED=1 CGO_CFLAGS="-fembed-bitcode" GOOS=ios GOARCH=$(arch) CC="$(clang_path) -isysroot $(sdk_path) -arch arm64 -miphoneos-version-min=11.0" go build -buildmode c-archive -trimpath -gcflags=all="-l" -ldflags="-w -s -X main.GitCommit=$(ver)" -o build/lib/ios/tonutils-proxy.a cmd/lib/main.go

# example: /home/user/android-ndk-r25c
ndk:=${NDK_PATH}
# example: linux-x86_64
ndk_arch:=${NDK_ARCH}
ndk_android_ver:=21

ndk:=${NDK_ROOT}
ndk_arch:=${NDK_ARCH}
ndk_android_ver:=21
ndk_cc:=$(ndk)/toolchains/llvm/prebuilt/$(ndk_arch)/bin/aarch64-linux-android$(ndk_android_ver)-clang

build-android-lib:
	CC=$(ndk_cc) CGO_ENABLED=1 GOOS=android GOARCH=arm64 go build -buildmode c-shared -trimpath -gcflags=all="-l" -ldflags="-w -s -X main.GitCommit=$(ver)" -o build/lib/android/tonutils-proxy.so cmd/lib/main.go

