package s3

import (
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/codegangsta/cli"
)

var (
	parallel     int
	dryRun       bool
	delete       bool
	public       bool
	quiet        bool
	ignoreErrors bool
	acl          string
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

func validACL() bool {
	if acl != "" && !ValidACLs[acl] {
		fmt.Fprintln(os.Stderr, "acl should be one of: private, public-read, public-read-write, authenticated-read, bucket-owner-read, bucket-owner-full-control, log-delivery-write")
		return false
	}
	return true
}

func Main(conn S3er, args []string, output io.Writer) int {
	out = output
	exitCode := 0

	getConnection := func(c *cli.Context) S3er {
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
		Destination: &delete,
	}

	app := cli.NewApp()
	app.Name = "s3"
	app.Usage = "S3 utility knife"
	app.Version = "0.0.1"
	app.Flags = commonFlags
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
				catKeys(conn, c.Args())
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
				getKeys(conn, c.Args())
			},
		},
		{
			Name:      "grep",
			Usage:     "Grep keys",
			ArgsUsage: "string key ...",
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					cli.ShowCommandHelp(c, "grep")
					exitCode = 1
					return
				}
				conn := getConnection(c)
				grepKeys(conn, c.Args())
			},
		},
		{
			Name:      "ls",
			Usage:     "List buckets or keys",
			ArgsUsage: "[bucket]",
			Action: func(c *cli.Context) {
				if len(c.Args()) < 1 {
					conn := getConnection(c)
					listBuckets(conn)
				} else {
					conn := getConnection(c)
					listKeys(conn, c.Args())
				}
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
				putBuckets(conn, c.Args())
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
				putKeys(conn, c.Args())
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
				rmBuckets(conn, c.Args())
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
				rmKeys(conn, c.Args())
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
				syncFiles(conn, c.Args())
			},
		},
	}
	app.Run(args)
	return exitCode
}
