ver := $(shell git describe --tags --always --dirty)

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

apple_clang_arch = $(if $(filter $(1),arm64),arm64,x86_64)
apple_headers_path = build/lib/apple/$(1)/include
apple_lib_path = build/lib/apple/$(1)/tonutils-proxy.a

define build_apple_variant
	GOOS=$(1) GOARCH=$(3) CGO_ENABLED=1 \
	CC="$(shell xcrun --sdk $(2) --find clang)" \
	CGO_CFLAGS="-isysroot $(shell xcrun --sdk $(2) --show-sdk-path) -arch $(call apple_clang_arch,$(3)) $(4) $(5)" \
	CGO_LDFLAGS="-isysroot $(shell xcrun --sdk $(2) --show-sdk-path) -arch $(call apple_clang_arch,$(3)) $(4) $(5)" \
	go build \
		-buildmode=c-archive -trimpath -gcflags=all="-l" \
		-ldflags="-w -s -X main.GitCommit=$(ver)" \
		-o "$(call apple_lib_path,$(6))" cmd/lib/main.go
	
	@mkdir -p $(call apple_headers_path,$(6)) 
	@mv -f build/lib/apple/$(6)/tonutils-proxy.h $(call apple_headers_path,$(6))/tonutils-proxy.h
endef

define fat_apple_variants
	@mkdir -p $(dir $(1))
	xcrun lipo -create $(2) -output $(1)
endef

build-apple-xcframework:
	# ios
	$(call build_apple_variant,ios,iphoneos,arm64,-mios-version-min=11.0,,ios-arm64) $ # arm64
	$(call build_apple_variant,ios,iphonesimulator,arm64,-mios-simulator-version-min=11.0,,ios-arm64-simulator) # arm64-simulator
	$(call build_apple_variant,ios,iphonesimulator,amd64,-mios-simulator-version-min=11.0,,ios-x86_64-simulator) # x86_64-simulator

	# macoOS
	$(call build_apple_variant,darwin,macosx,arm64,-mmacosx-version-min=12.0,,macos-arm64) # arm64
	$(call build_apple_variant,darwin,macosx,amd64,-mmacosx-version-min=12.0,,macos-x86_64) # x86-64
	$(call build_apple_variant,ios,macosx,arm64,-mios-version-min=13.1,-target arm64-apple-ios13.1-macabi,maccatalyst-arm64) # arm64-catalyst
	$(call build_apple_variant,ios,macosx,amd64,-mios-version-min=13.1,-target x86_64-apple-ios13.1-macabi,maccatalyst-x86_64) # x86-64-catalyst

	# join
	$(call fat_apple_variants,$(call apple_lib_path,ios-simulator),$(call apple_lib_path,ios-arm64-simulator) $(call apple_lib_path,ios-x86_64-simulator))
	@cp -Rf $(call apple_headers_path,ios-arm64-simulator) $(call apple_headers_path,ios-simulator)
	$(call fat_apple_variants,$(call apple_lib_path,macos),$(call apple_lib_path,macos-arm64) $(call apple_lib_path,macos-x86_64))
	@cp -Rf $(call apple_headers_path,macos-arm64) $(call apple_headers_path,macos)
	$(call fat_apple_variants,$(call apple_lib_path,maccatalyst),$(call apple_lib_path,maccatalyst-arm64) $(call apple_lib_path,maccatalyst-x86_64))
	@cp -Rf $(call apple_headers_path,maccatalyst-arm64) $(call apple_headers_path,maccatalyst)

	xcodebuild -create-xcframework \
		-library $(call apple_lib_path,ios-arm64) -headers $(call apple_headers_path,ios-arm64) \
		-library $(call apple_lib_path,ios-simulator) -headers $(call apple_headers_path,ios-simulator) \
		-library $(call apple_lib_path,macos) -headers $(call apple_headers_path,macos) \
		-library $(call apple_lib_path,maccatalyst) -headers $(call apple_headers_path,maccatalyst) \
		-output build/lib/apple/tonutils-proxy.xcframework

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

