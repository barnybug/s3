export GO15VENDOREXPERIMENT=1

package = github.com/barnybug/s3/cmd/s3
buildargs = -ldflags '-X github.com/barnybug/s3.version=${TRAVIS_TAG}'

.PHONY: release

default: install

deps:
	go get -d -v ./...

build-deps:
	go get github.com/pwaller/goupx
	go get github.com/gucumber/gucumber/cmd/gucumber

test: deps build-deps
	gucumber

install:
	go install -v ./cmd/s3

release:
	mkdir -p release
	GOOS=linux GOARCH=386 go build $(buildargs) -o release/s3-linux-386 $(package)
	GOOS=linux GOARCH=amd64 go build $(buildargs) -o release/s3-linux-amd64 $(package)
	GOOS=linux GOARCH=arm go build $(buildargs) -o release/s3-linux-arm $(package)
	GOOS=linux GOARCH=arm64 go build $(buildargs) -o release/s3-linux-arm64 $(package)
	GOOS=darwin GOARCH=amd64 go build $(buildargs) -o release/s3-darwin-amd64 $(package)
	GOOS=windows GOARCH=386 go build $(buildargs) -o release/s3-windows-386.exe $(package)
	GOOS=windows GOARCH=amd64 go build $(buildargs) -o release/s3-windows-amd64.exe $(package)
	goupx release/s3-linux-amd64
	upx release/s3-linux-386 release/s3-linux-arm release/s3-windows-386.exe
