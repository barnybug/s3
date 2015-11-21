package main

import (
	"os"

	"github.com/barnybug/s3"
)

func main() {
	exitCode := s3.Main(os.Args)
	os.Exit(exitCode)
}
