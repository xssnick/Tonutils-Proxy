echo "Building ARM64"
CGO_ENABLED=1 wails build -clean -platform darwin/arm64 &&
gon ./build/mac_build/arm/config.json
echo "ARM64 Done!"
echo "Press ENTER to compile AMD64..."
read -r -n 1 -s
echo "Building AMD64"
CGO_ENABLED=1 wails build -clean -platform darwin/amd64 &&
gon ./build/mac_build/amd/config.json