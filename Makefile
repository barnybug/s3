package = github.com/barnybug/s3/cmd/s3

.PHONY: release

default: install

test:
	gucumber

install:
	go install -v ./cmd/s3

deps:
	go get -d -v ./...

build-deps:
	go get github.com/lsegal/gucumber/cmd/gucumber

release: deps build-deps test
	mkdir -p release
	GOOS=linux GOARCH=386 go build -o release/s3-linux-386 $(package)
	GOOS=linux GOARCH=amd64 go build -o release/s3-linux-amd64 $(package)
	GOOS=linux GOARCH=arm go build -o release/s3-linux-arm $(package)
	GOOS=darwin GOARCH=amd64 go build -o release/s3-darwin-amd64 $(package)
	GOOS=windows GOARCH=386 go build -o release/s3-windows-386.exe $(package)
	GOOS=windows GOARCH=amd64 go build -o release/s3-windows-amd64.exe $(package)
	goupx release/s3-linux-amd64
	upx release/s3-linux-386 release/s3-linux-arm release/s3-windows-386.exe
