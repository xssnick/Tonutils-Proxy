echo "Building MAC GUI ARM"
mkdir -p build/gui/mac/arm/Tonutils\ Proxy.app/Contents/MacOS
CGO_CFLAGS=-mmacosx-version-min=10.9 CGO_LDFLAGS=-mmacosx-version-min=10.9 CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags '-w -s' -o build/gui/mac/arm/Tonutils\ Proxy.app/Contents/MacOS/tonutils-proxy cmd/proxy-gui/main.go

echo "Building MAC GUI AMD"
mkdir -p build/gui/mac/amd/Tonutils\ Proxy.app/Contents/MacOS
CGO_CFLAGS=-mmacosx-version-min=10.9 CGO_LDFLAGS=-mmacosx-version-min=10.9 CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags '-w -s' -o build/gui/mac/amd/Tonutils\ Proxy.app/Contents/MacOS/tonutils-proxy cmd/proxy-gui/main.go

echo "Building WINDOWS GUI AMD"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o build/tonutils-proxy-gui.exe cmd/proxy-gui/main.go
echo "Building LINUX GUI AMD"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/tonutils-proxy-gui cmd/proxy-gui/main.go

mkdir -p build/cli
echo "Building MAC CLI ARM"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o build/cli/tonutils-proxy-cli-darwin-arm64 cmd/proxy-cli/main.go
echo "Building MAC CLI AMD"
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/cli/tonutils-proxy-cli-darwin-amd64 cmd/proxy-cli/main.go
echo "Building LINUX CLI AMD"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/cli/tonutils-proxy-cli-linux-amd64 cmd/proxy-cli/main.go
echo "Building WINDOWS CLI AMD"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/cli/tonutils-proxy-cli-windows-amd64.exe cmd/proxy-cli/main.go