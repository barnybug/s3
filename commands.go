//
// s3 - Swiss army pen-knife for Amazon S3.
//
//   https://github.com/barnybug/s3
//
// Copyright (c) 2014 Barnaby Gray

package s3

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

var reBucketPath = regexp.MustCompile("^(?:s3://)?([^/]+)/?(.*)$")
var out io.Writer = os.Stdout

func extractBucketPath(url string) (string, string) {
	parts := reBucketPath.FindStringSubmatch(url)
	return parts[1], parts[2]
}

func listBuckets(conn s3iface.S3API) {
	output, err := conn.ListBuckets(nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, b := range output.Buckets {
		fmt.Fprintf(out, "s3://%s/\n", *b.Name)
	}
}

func iterateKeys(conn s3iface.S3API, urls []string, callback func(file File)) {
	for _, url := range urls {
		fs := getFilesystem(conn, url)
		ch := fs.Files()
		for file := range ch {
			callback(file)
		}
	}
}

func iterateKeysParallel(conn s3iface.S3API, urls []string, callback func(file File)) {
	// create pool for processing
	wg := sync.WaitGroup{}
	q := make(chan File, 1000)
	for i := 0; i < parallel; i += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range q {
				callback(key)
			}
		}()
	}

	iterateKeys(conn, urls, func(file File) {
		q <- file
	})

	close(q)
	wg.Wait()
}

func listKeys(conn s3iface.S3API, urls []string) {
	var count, totalSize int64
	iterateKeys(conn, urls, func(file File) {
		if quiet {
			fmt.Fprintln(out, file)
		} else {
			fmt.Fprintf(out, "%s\t%db\n", file, file.Size())
		}
		count += 1
		totalSize += file.Size()
	})
	if !quiet {
		fmt.Fprintf(out, "\n%d files, %d bytes\n", count, totalSize)
	}
}

func getKeys(conn s3iface.S3API, urls []string) {
	iterateKeysParallel(conn, urls, func(file File) {
		reader, err := file.Reader()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		// write files under relative path to the source path
		fpath := file.Relative()
		dirpath := path.Dir(fpath)
		if dirpath != "." {
			err = os.MkdirAll(dirpath, 0777)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		writer, err := os.Create(fpath)
		if err != nil {
			log.Fatal(err.Error())
		}
		nbytes, err := io.Copy(writer, reader)
		if err != nil {
			log.Fatal(err.Error())
		}
		if !quiet {
			fmt.Fprintf(out, "%s -> %s (%d bytes)\n", file, fpath, nbytes)
		}
	})
}

func catKeys(conn s3iface.S3API, urls []string) {
	iterateKeysParallel(conn, urls, func(file File) {
		reader, err := file.Reader()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		if strings.HasSuffix(file.String(), ".gz") {
			reader, err = gzip.NewReader(reader)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		_, err = io.Copy(out, reader)
		if err != nil {
			log.Fatal(err.Error())
		}
	})
}

func grepKeys(conn s3iface.S3API, args []string) {
	find := args[0]
	findBytes := []byte(find)
	urls := args[1:]
	iterateKeysParallel(conn, urls, func(file File) {
		reader, err := file.Reader()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		if strings.HasSuffix(file.String(), ".gz") {
			reader, err = gzip.NewReader(reader)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		buf := make([]byte, 4096)
		offset := 0
		for n, err := reader.Read(buf[offset:]); n > 0 && err == nil; n, err = reader.Read(buf) {
			if bytes.Contains(buf, findBytes) {
				fmt.Fprintln(out, file.String())
				break
			}
			// handle overlapping matches - copy last N-1 bytes to start of next
			offset = len(findBytes) - 1
			if offset > n {
				offset = n
			} else {
				copy(buf, buf[n-offset:])
			}
		}
		if err != nil && err != io.EOF {
			log.Fatal(err.Error())
		}
	})
}

func deleteBatch(conn s3iface.S3API, bucket string, batch []*s3.ObjectIdentifier) {
	if !dryRun {
		deleteRequest := s3.Delete{
			Objects: batch,
		}
		input := s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &deleteRequest,
		}
		conn.DeleteObjects(&input)
	}
}

func rmKeys(conn s3iface.S3API, urls []string) {
	batch := make([]*s3.ObjectIdentifier, 0, 1000)
	var bucket string
	start := time.Now()
	var deleted int
	iterateKeys(conn, urls, func(file File) {
		deleted += 1
		if !quiet {
			fmt.Fprintf(out, "D %s\n", file)
		}
		switch t := file.(type) {
		case *S3File:
			// optimize as a batch delete
			if t.bucket != bucket && len(batch) > 0 {
				deleteBatch(conn, bucket, batch)
				batch = batch[:0]
			}
			bucket = t.bucket
			obj := s3.ObjectIdentifier{Key: t.object.Key}
			batch = append(batch, &obj)
			if len(batch) == 1000 {
				deleteBatch(conn, bucket, batch)
				batch = batch[:0]
			}

		default:
			if !dryRun {
				file.Delete()
			}
		}
	})

	// final batch
	if len(batch) > 0 {
		deleteBatch(conn, bucket, batch)
	}
	end := time.Now()
	took := end.Sub(start)
	summary(0, deleted, 0, 0, took)
}

func rmBuckets(conn s3iface.S3API, urls []string) {
	for _, url := range urls {
		bucket, _ := extractBucketPath(url)
		input := s3.DeleteBucketInput{Bucket: aws.String(bucket)}
		_, err := conn.DeleteBucket(&input)
		if err != nil {
			log.Fatalln(err.Error())
		}
	}
}

func summary(added, deleted, updated, unchanged int, took time.Duration) {
	rate := float64(added+deleted+updated) / took.Seconds()

	if dryRun {
		fmt.Fprintln(out, "-- summary (dry-run) --")
	} else {
		fmt.Fprintln(out, "-- summary --")
	}
	fmt.Fprintf(out, `%d added %d deleted %d updated %d unchanged
took: %s (%.1f ops/s)

`, added, deleted, updated, unchanged, took, rate)
}

func putBuckets(conn s3iface.S3API, urls []string) {
	for _, url := range urls {
		input := s3.CreateBucketInput{
			ACL:    aws.String(acl),
			Bucket: aws.String(url),
		}
		_, err := conn.CreateBucket(&input)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

func putKeys(conn s3iface.S3API, urls []string) {
	sources := urls[:len(urls)-1]
	destination := urls[len(urls)-1]
	start := time.Now()
	dfs := getFilesystem(conn, destination)
	var added int
	iterateKeysParallel(conn, sources, func(file File) {
		reader, err := file.Reader()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		if !quiet {
			fmt.Fprintf(out, "A %s\n", file)
		}
		err = dfs.Create(file)
		if err != nil {
			log.Fatal(err.Error())
		}
		added += 1
	})
	end := time.Now()
	took := end.Sub(start)
	summary(added, 0, 0, 0, took)
}

type File interface {
	Relative() string
	Size() int64
	MD5() []byte
	Reader() (io.ReadCloser, error)
	Delete() error
	String() string
	IsDirectory() bool
}

type Filesystem interface {
	Files() <-chan File
	Create(src File) error
	Delete(path string) error
}

func getFilesystem(conn s3iface.S3API, url string) Filesystem {
	if strings.HasPrefix(url, "s3:") {
		bucket, prefix := extractBucketPath(url)
		return &S3Filesystem{conn: conn, bucket: bucket, path: prefix}
	} else {
		return &LocalFilesystem{path: url}
	}
}

type Action struct {
	Action string
	File   File
}

func processAction(action Action, fs2 Filesystem) {
	switch action.Action {
	case "create":
		if !quiet {
			fmt.Fprintf(out, "A %s\n", action.File.Relative())
		}
		if dryRun {
			return
		}
		err := fs2.Create(action.File)
		if err != nil {
			if ignoreErrors {
				fmt.Fprintf(out, "E %s: %s\n", action.File.Relative(), err)
			} else {
				log.Fatal(err)
			}
		}
	case "delete":
		if !quiet {
			fmt.Fprintf(out, "D %s\n", action.File.Relative())
		}
		if dryRun {
			return
		}
		err := fs2.Delete(action.File.Relative())
		if err != nil {
			log.Fatal(err)
		}
	case "update":
		if !quiet {
			fmt.Fprintf(out, "U %s\n", action.File.Relative())
		}
		if dryRun {
			return
		}
		err := fs2.Create(action.File)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func syncFiles(conn s3iface.S3API, urls []string) {
	if len(urls) != 2 {
		// TODO: support multiple sources -> single destination
		panic("Unsupported")
	}
	url1 := urls[0]
	url2 := urls[1]
	start := time.Now()
	fs1 := getFilesystem(conn, url1)
	fs2 := getFilesystem(conn, url2)
	ch1 := fs1.Files()
	f1 := <-ch1

	ch2 := fs2.Files()
	f2 := <-ch2

	// create pool for processing
	wg := sync.WaitGroup{}
	q := make(chan Action, 1000)
	for i := 0; i < parallel; i += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for action := range q {
				processAction(action, fs2)
			}
		}()
	}

	var added, deleted, updated, unchanged int
	for {
		// iterate files in fs1 and fs2
		// if f1 is nil and f2 is nil, we're done
		// if f1 is nil or f1 < f2, create f1
		// if f2 is nil or f1 > f2, delete f2
		// if f1 = f2, check size, md5
		if f1 == nil && f2 == nil {
			break
		} else if f2 == nil || (f1 != nil && f1.Relative() < f2.Relative()) {
			q <- Action{"create", f1}
			added += 1
			f1 = <-ch1
		} else if f1 == nil || (f2 != nil && f1.Relative() > f2.Relative()) {
			if deleteExtra {
				q <- Action{"delete", f2}
				deleted += 1
			}
			f2 = <-ch2
		} else if f1.Size() != f2.Size() || bytes.Compare(f1.MD5(), f2.MD5()) != 0 {
			q <- Action{"update", f1}
			updated += 1
			f1 = <-ch1
			f2 = <-ch2
		} else {
			unchanged += 1
			f1 = <-ch1
			f2 = <-ch2
		}
	}

	close(q)
	wg.Wait()

	end := time.Now()
	took := end.Sub(start)
	summary(added, deleted, updated, unchanged, took)
}
