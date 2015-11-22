package s3

import (
	"crypto/md5"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type LocalFilesystem struct {
	err  error
	path string
}

func (self *LocalFilesystem) Error() error {
	return self.err
}

func scanFiles(ch chan<- File, fullpath string, relpath string) error {
	entries, err := ioutil.ReadDir(fullpath)
	if os.IsNotExist(err) {
		// this is fine - indicates no files are there
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		f := filepath.Join(fullpath, entry.Name())
		r := filepath.Join(relpath, entry.Name())
		if entry.IsDir() {
			// recurse
			err := scanFiles(ch, f, r)
			if err != nil {
				return err
			}
		} else {
			ch <- &LocalFile{entry, f, r, nil}
		}
	}
	return nil
}

func (self *LocalFilesystem) Files() <-chan File {
	ch := make(chan File, 1000)

	// use relative path to file or directory:
	// path/to/file -> file
	// parent/path -> path
	// path/ -> ''
	ps := strings.Split(self.path, "/")
	relpath := ps[len(ps)-1]
	go func() {
		defer close(ch)
		fi, err := os.Stat(self.path)
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			self.err = err
			return
		}
		if fi.IsDir() {
			err := scanFiles(ch, self.path, relpath)
			if err != nil {
				self.err = err
			}
		} else {
			ch <- &LocalFile{fi, self.path, relpath, nil}
		}
	}()
	return ch
}

func (self *LocalFilesystem) Create(src File) error {
	reader, err := src.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()
	fullpath := filepath.Join(self.path, src.Relative())
	if src.IsDirectory() {
		err = os.MkdirAll(fullpath, 0777)
	} else {
		// create containing directory
		dirpath := filepath.Dir(fullpath)
		err = os.MkdirAll(dirpath, 0777)
		if err != nil {
			return err
		}
		writer, err := os.Create(fullpath)
		if err != nil {
			return err
		}
		defer writer.Close()
		_, err = io.Copy(writer, reader)
	}
	return err
}

func (self *LocalFilesystem) Delete(path string) error {
	fullpath := filepath.Join(self.path, path)
	return os.Remove(fullpath)
}

type LocalFile struct {
	info     os.FileInfo
	fullpath string
	relpath  string
	md5      []byte
}

func (self *LocalFile) Relative() string {
	return self.relpath
}

func (self *LocalFile) Size() int64 {
	return self.info.Size()
}

func (self *LocalFile) IsDirectory() bool {
	return false
}

func (self *LocalFile) MD5() []byte {
	if self.md5 == nil {
		// cache md5
		h := md5.New()
		reader, err := os.Open(self.fullpath)
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.Copy(h, reader)
		if err != nil {
			log.Fatal(err)
		}
		self.md5 = h.Sum(nil)
	}
	return self.md5
}

func (self *LocalFile) Reader() (io.ReadCloser, error) {
	return os.Open(self.fullpath)
}

func (self *LocalFile) Delete() error {
	return os.Remove(self.fullpath)
}

func (self *LocalFile) String() string {
	return self.relpath
}
