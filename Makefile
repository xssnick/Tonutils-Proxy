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