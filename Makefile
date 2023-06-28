ver := $(shell git log -1 --pretty=format:%h)

compile:
	mkdir -p build/cli
	echo "Building MAC CLI ARM"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-darwin-arm64 cmd/proxy-cli/main.go
	echo "Building MAC CLI AMD"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-darwin-amd64 cmd/proxy-cli/main.go
	echo "Building LINUX CLI AMD"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-linux-amd64 cmd/proxy-cli/main.go
	echo "Building WINDOWS CLI AMD"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.GitCommit=$(ver)" -o build/cli/tonutils-proxy-cli-windows-amd64.exe cmd/proxy-cli/main.go

sdk := iphoneos
arch := arm64
sdk_path := $(shell xcrun --sdk ${sdk} --show-sdk-path)
clang_path := $(shell xcrun --sdk ${sdk} --find clang)
goRoot := $(shell go env GOROOT)

build-ios-lib:
	SDK=$(sdk) CGO_ENABLED=1 CGO_CFLAGS="-fembed-bitcode" GOOS=ios GOARCH=$(arch) CC=$(goRoot)/misc/ios/clangwrap.sh go build -buildmode c-archive -trimpath -gcflags=all="-l -B" -ldflags="-w -s" -o build/lib/ios/tonutils-proxy.a lib/main.go

ndk := ${NDK_PATH} # example: /home/user/android-ndk-r25c
ndk_arch := ${NDK_ARCH} # example: linux-x86_64
ndk_android_ver := 21

build-android-lib:
	CC=$(ndk)/toolchains/llvm/prebuilt/$(ndk_arch)/bin/aarch64-linux-android$(ndk_android_ver)-clang CGO_ENABLED=1 GOOS=android GOARCH=arm64 go build -buildmode c-archive -trimpath -gcflags=all="-l -B" -ldflags="-w -s" -o build/lib/android/tonutils-proxy.a lib/main.go
