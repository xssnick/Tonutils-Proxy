echo "Building MAC GUI ARM"
mkdir -p build/TonutilsProxyARM.app/Contents/MacOS
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o build/TonutilsProxyARM.app/Contents/MacOS/tonutils-proxy cmd/proxy-gui/main.go

echo "Building MAC GUI AMD"
mkdir -p build/TonutilsProxyAMD.app/Contents/MacOS
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o build/TonutilsProxyAMD.app/Contents/MacOS/tonutils-proxy cmd/proxy-gui/main.go

echo "Building WINDOWS GUI AMD"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o build/tonutils-proxy-gui.exe cmd/proxy-gui/main.go
echo "Building LINUX GUI AMD"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/tonutils-proxy-gui cmd/proxy-gui/main.go

mkdir -p build/cli
echo "Building MAC CLI ARM"
GOOS=darwin GOARCH=arm64 go build -o build/cli/tonutils-proxy-cli-darwin-arm64 cmd/proxy-cli/main.go
echo "Building MAC CLI AMD"
GOOS=darwin GOARCH=amd64 go build -o build/cli/tonutils-proxy-cli-darwin-amd64 cmd/proxy-cli/main.go
echo "Building LINUX CLI AMD"
GOOS=linux GOARCH=amd64 go build -o build/cli/tonutils-proxy-cli-linux-amd64 cmd/proxy-cli/main.go
echo "Building WINDOWS CLI AMD"
GOOS=windows GOARCH=amd64 go build -o build/cli/tonutils-proxy-cli-windows-amd64.exe cmd/proxy-cli/main.go

open build/TonutilsProxyARM.app