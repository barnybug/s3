package main

import (
	"os"

	"github.com/madedotcom/s3/tree/grr-version-api"
)

func main() {
	exitCode := s3.Main(nil, os.Args, os.Stdout)
	os.Exit(exitCode)
}
