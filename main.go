package s3

import (
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/codegangsta/cli"
)

var (
	parallel     int
	dryRun       bool
	deleteExtra  bool
	public       bool
	quiet        bool
	ignoreErrors bool
	acl          string
)
var version = "master" /* passed in by go build */

var ValidACLs = map[string]bool{
	"private":                   true,
	"public-read":               true,
	"public-read-write":         true,
	"authenticated-read":        true,
	"bucket-owner-read":         true,
	"bucket-owner-full-control": true,
	"log-delivery-write":        true,
}

func validACL() bool {
	if acl != "" && !ValidACLs[acl] {
		fmt.Fprintln(os.Stderr, "acl should be one of: private, public-read, public-read-write, authenticated-read, bucket-owner-read, bucket-owner-full-control, log-delivery-write")
		return false
	}
	return true
}

func Main(conn s3iface.S3API, args []string, output io.Writer) int {
	out = output
	exitCode := 0

	checkErr := func(err error) {
		if err != nil {
			fmt.Fprintf(out, "Error: %s\n", err)
			exitCode = 1
		}
	}

	getConnection := func(c *cli.Context) s3iface.S3API {
		if conn == nil {
			region := c.Parent().String("region")
			config := aws.Config{
				Region: aws.String(region),
			}
			conn = s3.New(session.New(), &config)
		}
		return conn
	}

	commonFlags := []cli.Flag{
		cli.IntFlag{
			Name:        "p",
			Value:       32,
			Usage:       "number of parallel operations to run",
			Destination: &parallel,
		},
		cli.BoolFlag{
			Name:        "n",
			Usage:       "dry-run, no actions taken",
			Destination: &dryRun,
		},
		cli.BoolFlag{
			Name:        "ignore-errors",
			Usage:       "",
			Destination: &ignoreErrors,
		},
		cli.BoolFlag{
			Name:        "q",
			Usage:       "",
			Destination: &quiet,
		},
		cli.StringFlag{
			Name:   "region",
			Usage:  "set region, otherwise environment variable AWS_REGION is checked, finally defaulting to us-east-1",
			Value:  "us-east-1",
			EnvVar: "AWS_REGION",
		},
	}

	aclFlag := cli.StringFlag{
		Name:        "acl",
		Usage:       "set acl to one of: private, public-read, public-read-write, authenticated-read, bucket-owner-read, bucket-owner-full-control, log-delivery-write",
		Destination: &acl,
	}
	publicFlag := cli.BoolFlag{
		Name:        "public, P",
		Usage:       "",
		Destination: &public,
	}
	deleteFlag := cli.BoolFlag{
		Name:        "delete",
		Usage:       "delete extraneous files from destination",
		Destination: &deleteExtra,
	}

	app := cli.NewApp()
	app.Name = "s3"
	app.Usage = "S3 utility knife"
	app.Version = version
	app.Flags = commonFlags
	app.Writer = out
	app.Commands = []cli.Command{
		{
			Name:      "cat",
			Usage:     "Cat key contents",
			ArgsUsage: "key ...",
			Flags:     commonFlags,
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					cli.ShowCommandHelp(c, "cat")
					exitCode = 1
					return
				}
				conn := getConnection(c)
				err := catKeys(conn, c.Args())
				checkErr(err)
			},
		},
		{
			Name:      "get",
			Usage:     "Download keys",
			ArgsUsage: "key ...",
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					cli.ShowCommandHelp(c, "get")
					exitCode = 1
					return
				}
				conn := getConnection(c)
				err := getKeys(conn, c.Args())
				checkErr(err)
			},
		},
		{
			Name:      "grep",
			Usage:     "Grep keys",
			ArgsUsage: "string key ...",
			Action: func(c *cli.Context) {
				if len(c.Args()) < 2 {
					cli.ShowCommandHelp(c, "grep")
					exitCode = 1
					return
				}
				conn := getConnection(c)
				find := c.Args().First()
				urls := c.Args().Tail()
				err := grepKeys(conn, find, urls)
				checkErr(err)
			},
		},
		{
			Name:      "ls",
			Usage:     "List buckets or keys",
			ArgsUsage: "[bucket]",
			Action: func(c *cli.Context) {
				var err error
				if len(c.Args()) < 1 {
					conn := getConnection(c)
					err = listBuckets(conn)
				} else {
					conn := getConnection(c)
					err = listKeys(conn, c.Args())
				}
				checkErr(err)
			},
		},
		{
			Name:      "mb",
			Usage:     "Create bucket",
			ArgsUsage: "bucket",
			Action: func(c *cli.Context) {
				if len(c.Args()) != 1 {
					cli.ShowCommandHelp(c, "mb")
					exitCode = 1
					return
				}
				conn := getConnection(c)
				err := putBuckets(conn, c.Args())
				checkErr(err)
			},
		},
		{
			Name:      "put",
			Usage:     "Upload files",
			ArgsUsage: "source [source ...] dest",
			Flags:     []cli.Flag{aclFlag, publicFlag},
			Action: func(c *cli.Context) {
				if len(c.Args()) < 2 {
					cli.ShowCommandHelp(c, "put")
					exitCode = 1
					return
				}
				if public {
					acl = "public-read"
				}
				if !validACL() {
					exitCode = 1
					return
				}
				conn := getConnection(c)
				args := c.Args()
				sources := args[:len(args)-1]
				destination := args[len(args)-1]
				err := putKeys(conn, sources, destination)
				checkErr(err)
			},
		},
		{
			Name:      "rb",
			Usage:     "Remove bucket(s)",
			ArgsUsage: "bucket ...",
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					cli.ShowCommandHelp(c, "rb")
					exitCode = 1
					return
				}
				conn := getConnection(c)
				err := rmBuckets(conn, c.Args())
				checkErr(err)
			},
		},
		{
			Name:      "rm",
			Usage:     "Remove keys",
			ArgsUsage: "key ...",
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					cli.ShowCommandHelp(c, "rm")
					exitCode = 1
					return
				}
				conn := getConnection(c)
				err := rmKeys(conn, c.Args())
				checkErr(err)
			},
		},
		{
			Name:      "sync",
			Usage:     "Synchronise local to s3, s3 to s3 or s3 to local",
			ArgsUsage: "source dest",
			Flags:     []cli.Flag{aclFlag, publicFlag, deleteFlag},
			Action: func(c *cli.Context) {
				if len(c.Args()) != 2 {
					cli.ShowCommandHelp(c, "sync")
					exitCode = 1
					return
				}
				if public {
					acl = "public-read"
				}
				if !validACL() {
					exitCode = 1
					return
				}
				conn := getConnection(c)
				err := syncFiles(conn, c.Args()[0], c.Args()[1])
				checkErr(err)
			},
		},
	}
	app.Run(args)
	return exitCode
}
