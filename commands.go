//
// s3 - Swiss army pen-knife for Amazon S3.
//
//   https://github.com/barnybug/s3
//
// Copyright (c) 2015 Barnaby Gray

package s3

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
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
var err io.Writer = os.Stderr

var (
	ErrNotFound = errors.New("No files found")
)

func extractBucketPath(url string) (string, string) {
	parts := reBucketPath.FindStringSubmatch(url)
	return parts[1], parts[2]
}

func listBuckets(conn s3iface.S3API) error {
	output, err := conn.ListBuckets(nil)
	if err != nil {
		return err
	}
	for _, b := range output.Buckets {
		fmt.Fprintf(out, "s3://%s/\n", *b.Name)
	}
	return nil
}

func iterateKeys(conn s3iface.S3API, urls []string, callback func(file File) error) error {
	found := false
	for _, url := range urls {
		fs := getFilesystem(conn, url)
		ch := fs.Files()
		for file := range ch {
			found = true
			err := callback(file)
			if err != nil {
				return err
			}
		}
		if fs.Error() != nil {
			return fs.Error()
		}
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func iterateKeysParallel(conn s3iface.S3API, urls []string, callback func(file File) error) error {
	// create pool for processing
	var err error
	wg := sync.WaitGroup{}
	q := make(chan File, 1000)
	for i := 0; i < parallel; i += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range q {
				e := callback(key)
				if e != nil {
					err = e
					return
				}
			}
		}()
	}

	e := iterateKeys(conn, urls, func(file File) error {
		q <- file
		return nil
	})
	if e != nil {
		return e
	}

	close(q)
	wg.Wait()
	return err
}

func listKeys(conn s3iface.S3API, urls []string) error {
	var count, totalSize int64
	err := iterateKeys(conn, urls, func(file File) error {
		if quiet {
			fmt.Fprintln(out, file)
		} else {
			fmt.Fprintf(out, "%s\t%db\n", file, file.Size())
		}
		count += 1
		totalSize += file.Size()
		return nil
	})
	if err != nil && err != ErrNotFound {
		return err
	}
	if !quiet {
		fmt.Fprintf(out, "\n%d files, %d bytes\n", count, totalSize)
	}
	return nil
}

func getKeys(conn s3iface.S3API, urls []string) error {
	for _, url := range urls {
		if !isS3Url(url) {
			return errors.New("s3:// url required")
		}
	}

	err := iterateKeysParallel(conn, urls, func(file File) error {
		reader, err := file.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()

		// write files under relative path to the source path
		fpath := file.Relative()
		dirpath := path.Dir(fpath)
		if dirpath != "." {
			err = os.MkdirAll(dirpath, 0777)
			if err != nil {
				return err
			}
		}

		writer, err := os.Create(fpath)
		if err != nil {
			return err
		}
		nbytes, err := io.Copy(writer, reader)
		if err != nil {
			return err
		}
		if !quiet {
			fmt.Fprintf(out, "%s -> %s (%d bytes)\n", file, fpath, nbytes)
		}
		return nil
	})
	return err
}

func catKeys(conn s3iface.S3API, urls []string) error {
	return iterateKeysParallel(conn, urls, func(file File) error {
		reader, err := file.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()

		if strings.HasSuffix(file.String(), ".gz") {
			reader, err = gzip.NewReader(reader)
			if err != nil {
				return err
			}
		}

		_, err = io.Copy(out, reader)
		if err != nil {
			return err
		}
		return nil
	})
}

func outputMatches(buf []byte, needle []byte, prefix string) {
	p := 0
	for {
		i := bytes.Index(buf[p:], needle)
		if i == -1 {
			break
		}
		i += p
		lineStart := bytes.LastIndexByte(buf[:i], byte('\n'))
		if lineStart == -1 {
			lineStart = 0
		} else {
			lineStart += 1
		}
		lineEnd := bytes.IndexByte(buf[i:], byte('\n'))
		if lineEnd == -1 {
			lineEnd = len(buf)
		} else {
			lineEnd += i
		}
		line := string(buf[lineStart:lineEnd])

		fmt.Fprintf(out, "%s%s\n", prefix, line)

		p = lineEnd + 1
		if p > len(buf)-len(needle) {
			break
		}
	}
}

func grepKeys(conn s3iface.S3API, find string, urls []string, noKeysPrefix bool, keysWithMatches bool) error {
	needle := []byte(find)

	return iterateKeysParallel(conn, urls, func(file File) error {
		reader, err := file.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()

		if strings.HasSuffix(file.String(), ".gz") {
			reader, err = gzip.NewReader(reader)
			if err != nil {
				return err
			}
		}

		prefix := ""
		if !noKeysPrefix {
			prefix = file.String() + ":"
		}

		buf := make([]byte, 4096)
		offset := 0
		for n, err := reader.Read(buf[offset:]); n > 0 && err == nil; n, err = reader.Read(buf[offset:]) {
			if bytes.Contains(buf[:n+offset], needle) {
				if keysWithMatches {
					// only filename required, bail early
					fmt.Fprintln(out, file.String())
					break
				} else {
					outputMatches(buf[:n+offset], needle, prefix)
				}
			}
			// handle overlapping matches - copy last N-1 bytes to start of next
			offset = len(needle) - 1
			if offset > n {
				offset = n
			} else {
				copy(buf, buf[n-offset:])
			}
		}
		if err != nil && err != io.EOF {
			return err
		}
		return nil
	})
}

func deleteBatch(conn s3iface.S3API, bucket string, batch []*s3.ObjectIdentifier) error {
	if !dryRun {
		deleteRequest := s3.Delete{
			Objects: batch,
		}
		input := s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &deleteRequest,
		}
		_, err := conn.DeleteObjects(&input)
		return err
	}
	return nil
}

func rmKeys(conn s3iface.S3API, urls []string) error {
	for _, url := range urls {
		if !isS3Url(url) {
			return errors.New("Cowardly refusing to remove local files. Use rm.")
		}
	}
	batch := make([]*s3.ObjectIdentifier, 0, 1000)
	var bucket string
	start := time.Now()
	var deleted int
	err := iterateKeys(conn, urls, func(file File) error {
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
		return nil
	})
	if err != nil {
		return err
	}

	// final batch
	if len(batch) > 0 {
		deleteBatch(conn, bucket, batch)
	}
	end := time.Now()
	took := end.Sub(start)
	summary(0, deleted, 0, 0, took)
	return nil
}

func rmBuckets(conn s3iface.S3API, buckets []string) error {
	for _, name := range buckets {
		bucket, _ := extractBucketPath(name)
		input := s3.DeleteBucketInput{Bucket: aws.String(bucket)}
		_, err := conn.DeleteBucket(&input)
		if err != nil {
			return err
		}
	}
	return nil
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

func putBuckets(conn s3iface.S3API, buckets []string) error {
	for _, bucket := range buckets {
		input := s3.CreateBucketInput{
			ACL:    aws.String(acl),
			Bucket: aws.String(bucket),
		}
		_, err := conn.CreateBucket(&input)
		if err != nil {
			return err
		}
	}
	return nil
}

func putKeys(conn s3iface.S3API, sources []string, destination string) error {
	start := time.Now()
	if !isS3Url(destination) {
		return errors.New("s3:// url required for destination")
	}
	dfs := getFilesystem(conn, destination)
	var added int
	err := iterateKeysParallel(conn, sources, func(file File) error {
		reader, err := file.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()

		if !quiet {
			fmt.Fprintf(out, "A %s\n", file)
		}
		err = dfs.Create(file)
		if err != nil {
			return err
		}
		added += 1
		return nil
	})
	if err != nil {
		return err
	}
	end := time.Now()
	took := end.Sub(start)
	summary(added, 0, 0, 0, took)

	return nil
}

func isS3Url(url string) bool {
	return strings.HasPrefix(url, "s3:")
}

func getFilesystem(conn s3iface.S3API, url string) Filesystem {
	if isS3Url(url) {
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

func processAction(action Action, fs2 Filesystem) error {
	switch action.Action {
	case "create":
		if !quiet {
			fmt.Fprintf(out, "A %s\n", action.File.Relative())
		}
		if dryRun {
			return nil
		}
		err := fs2.Create(action.File)
		if err != nil {
			if ignoreErrors {
				fmt.Fprintf(out, "E %s: %s\n", action.File.Relative(), err)
			} else {
				return err
			}
		}
	case "delete":
		if !quiet {
			fmt.Fprintf(out, "D %s\n", action.File.Relative())
		}
		if dryRun {
			return nil
		}
		err := fs2.Delete(action.File.Relative())
		if err != nil {
			return err
		}
	case "update":
		if !quiet {
			fmt.Fprintf(out, "U %s\n", action.File.Relative())
		}
		if dryRun {
			return nil
		}
		err := fs2.Create(action.File)
		if err != nil {
			return err
		}
	}
	return nil
}

func syncFiles(conn s3iface.S3API, src, dest string) error {
	start := time.Now()
	fs1 := getFilesystem(conn, src)
	fs2 := getFilesystem(conn, dest)
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
	var err error
	for {
		err = fs1.Error()
		if err != nil {
			break
		}
		err = fs2.Error()
		if err != nil {
			break
		}
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
	if err != nil {
		return err
	}

	end := time.Now()
	took := end.Sub(start)
	summary(added, deleted, updated, unchanged, took)
	return nil
}
