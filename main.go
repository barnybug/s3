package s3

import (
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	parallel     int
	dryRun       bool
	delete       bool
	public       bool
	quiet        bool
	ignoreErrors bool
	acl          string
	region       string
)

var ValidACLs = map[string]bool{
	"private":                   true,
	"public-read":               true,
	"public-read-write":         true,
	"authenticated-read":        true,
	"bucket-owner-read":         true,
	"bucket-owner-full-control": true,
	"log-delivery-write":        true,
}

var minArgs = map[string]int{
	"cat":  1,
	"get":  1,
	"ls":   0,
	"mb":   1,
	"put":  2,
	"rb":   1,
	"rm":   1,
	"sync": 2,
}

func Run(conn S3er, args []string) {
	fs := flag.NewFlagSet("s3", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: s3 COMMAND [source...] [destination]

Commands:
	cat	Cat key contents
	get	Download keys
	ls	List buckets or keys
	mb 	Create bucket
	put 	Upload files
	rb	Remove bucket
	rm	Delete keys
	sync	Synchronise local to s3, s3 to s3 or s3 to local

Options:
`)
		fs.PrintDefaults()
	}
	fs.IntVar(&parallel, "p", 32, "number of parallel operations to run")
	fs.BoolVar(&dryRun, "n", false, "dry-run, no actions taken")
	fs.BoolVar(&delete, "delete", false, "delete extraneous files from destination (sync)")
	fs.BoolVar(&public, "P", false, "shortcut to set acl to public-read")
	fs.BoolVar(&ignoreErrors, "ignore-errors", false, "ignore errors writing files")
	fs.BoolVar(&quiet, "q", false, "quieter (less verbose) output")
	fs.StringVar(&acl, "acl", "", "set acl to one of: private, public-read, public-read-write, authenticated-read, bucket-owner-read, bucket-owner-full-control, log-delivery-write")
	fs.StringVar(&region, "region", "", "set region, without environment variables AWS_DEFAULT_REGION or EC2_REGION are checked, and otherwise defaults to us-east-1")

	if len(args) < 2 {
		fs.Usage()
		os.Exit(-1)
	}

	command := args[1]
	// pop out the command argument
	fs.Parse(args[2:])
	if public {
		acl = "public-read"
	}
	if acl != "" && !ValidACLs[acl] {
		fmt.Fprintln(os.Stderr, "-acl should be one of: private, public-read, public-read-write, authenticated-read, bucket-owner-read, bucket-owner-full-control, log-delivery-write")
		os.Exit(-1)
	}

	if conn == nil {
		config := aws.Config{}
		conn = s3.New(session.New(), &config)
	}

	if _, ok := minArgs[command]; !ok {
		fs.Usage()
		os.Exit(-1)
	}

	if len(fs.Args()) < minArgs[command] {
		fmt.Fprintln(os.Stderr, "More arguments required\n")
		fs.Usage()
		os.Exit(-1)
	}

	switch command {
	case "cat":
		catKeys(conn, fs.Args())
	case "get":
		getKeys(conn, fs.Args())
	case "ls":
		if len(fs.Args()) < 1 {
			listBuckets(conn)
		} else {
			listKeys(conn, fs.Args())
		}
	case "mb":
		putBuckets(conn, fs.Args())
	case "put":
		putKeys(conn, fs.Args())
	case "sync":
		syncFiles(conn, fs.Args())
	case "rm":
		rmKeys(conn, fs.Args())
	case "rb":
		rmBuckets(conn, fs.Args())
	}
}
