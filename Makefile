all: deps test install

test:
	go test ./...

install:
	go install -v ./...

deps:
	go get -d -v ./...
