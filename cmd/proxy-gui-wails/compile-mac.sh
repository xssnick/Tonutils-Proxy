echo "Building ARM64"
CGO_ENABLED=1 wails build -clean -platform darwin/arm64 &&
gon ./build/mac_build/arm/config.json

echo "Building AMD64"
CGO_ENABLED=1 wails build -clean -platform darwin/amd64 &&
gon ./build/mac_build/amd/config.json