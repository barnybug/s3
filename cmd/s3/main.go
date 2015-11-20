package main

import (
	"os"

	"github.com/barnybug/s3"
)

func main() {
	s3.Run(nil, os.Args)
}
