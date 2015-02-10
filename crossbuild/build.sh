#!/bin/bash -e

rm -rf build; mkdir build

# fetch dependencies
go get -d -v

GOOS=linux GOARCH=amd64 go build -v -o build/s3-linux-amd64
GOOS=linux GOARCH=386 go build -v -o build/s3-linux-386
GOOS=linux GOARCH=arm GOARM=5 go build -v -o build/s3-linux-arm5
GOOS=darwin GOARCH=amd64 go build -v -o build/s3-darwin-amd64
GOOS=darwin GOARCH=386 go build -v -o build/s3-darwin-386
GOOS=windows GOARCH=386 go build -v -o build/s3-windows-386.exe
GOOS=windows GOARCH=amd64 go build -v -o build/s3-windows-amd64.exe

# compress resulting executables
# workaround for bug (see: https://github.com/pwaller/goupx)
go get github.com/pwaller/goupx/
/go/bin/goupx -u=false build/s3-linux-amd64
upx build/*
chmod -R a+rw build
