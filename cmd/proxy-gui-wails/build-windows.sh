echo "Building Windows amd64"
CGO_ENABLED=1 wails build -platform windows/amd64 -nsis -webview2 download