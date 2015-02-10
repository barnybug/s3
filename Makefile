all: deps test install

test:
	go test ./...

install:
	go install -v ./...

deps:
	go get -d -v ./...

testdeps:
	go get -d -v github.com/motain/gocheck

release:
	docker build -t s3-crossbuild crossbuild
	docker run --rm -v "$(PWD)":/usr/src/myapp -w /usr/src/myapp s3-crossbuild
