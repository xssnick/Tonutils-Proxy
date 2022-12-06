echo "Building MAC GUI ARM"
mkdir -p build/gui/mac/arm/Tonutils\ Proxy.app/Contents/MacOS
CGO_CFLAGS=-mmacosx-version-min=10.9 CGO_LDFLAGS=-mmacosx-version-min=10.9 CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags '-w -s' -o build/gui/mac/arm/Tonutils\ Proxy.app/Contents/MacOS/tonutils-proxy cmd/proxy-gui/main.go

echo "Building MAC GUI AMD"
mkdir -p build/gui/mac/amd/Tonutils\ Proxy.app/Contents/MacOS
CGO_CFLAGS=-mmacosx-version-min=10.9 CGO_LDFLAGS=-mmacosx-version-min=10.9 CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags '-w -s' -o build/gui/mac/amd/Tonutils\ Proxy.app/Contents/MacOS/tonutils-proxy cmd/proxy-gui/main.go

echo "Building WINDOWS GUI AMD"
p=$(pwd)
# To start script need to declare your own LIBS env variable
#LIBS="<YOUR PATH>"
GO111MODULE="on" CGO_CXXFLAGS="-I$LIBS\webview2\build\native\include" CGO_LDFLAGS="-L$LIBS\webview2\build\native\x64" CGO_ENABLED=1 GOOS="windows" GOARCH="amd64"  go build -ldflags="-H windowsgui" -o build/gui/windows/tonutils-proxy-gui.exe cmd/proxy-gui/main.go
cp "$LIBS\webview2\build\native\x64\WebView2Loader.dll" "$p\build\gui\windows\WebView2Loader.dll"
rh.exe -open build/gui/windows/tonutils-proxy-gui.exe -save build/gui/windows/tonutils-proxy-gui.exe -action addskip -res resources/windows/ton_icon.ico -mask ICONGROUP,MAINICON,
cp "$p\resources\windows\win-install.iss" "$p\build\gui\windows\win-install.iss"
cp "$p\resources\windows\ton_icon.ico" "$p\build\gui\windows\ton_icon.ico"
iscc /Q[p] "$p\build\gui\windows\win-install.iss"
rm "$p\build\gui\windows\win-install.iss"
rm "$p\build\gui\windows\ton_icon.ico"

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