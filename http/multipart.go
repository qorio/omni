package http

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type PartCheck func(part int, formName, fileName string, peek *bufio.Reader) error
type PartSink func(part int, formName, fileName string) (io.WriteCloser, error)
type PartComplete func(part int, formName, fileName string, length int64)

var (
	ErrMissingInput error = errors.New("missing-input")
)

func FileSystemSink(rootDir string) PartSink {
	return func(part int, formName, fileName string) (dst io.WriteCloser, err error) {
		// Take either the form name or the filename
		id := formName
		if id == "" {
			id = fileName
		}
		if id == "" {
			return nil, ErrMissingInput
		}
		p := filepath.Join(rootDir, id)
		dst, err = os.Create(p)
		return
	}
}

func ProcessMultiPartUpload(resp http.ResponseWriter, req *http.Request,
	check PartCheck, sink PartSink, completion PartComplete) (err error) {

	//get the multipart reader for the request.
	reader, err := req.MultipartReader()
	if err != nil {
		return err
	}

	//copy each part to destination.
	for i := 0; ; i++ {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		// use bufio reader so we can peek
		buf_reader := bufio.NewReader(part)

		if check != nil {
			err := check(i, part.FormName(), part.FileName(), buf_reader)
			if err != nil {
				return err
			}
		}
		if sink != nil {
			dst, err := sink(i, part.FormName(), part.FileName())
			defer dst.Close()

			if err != nil {
				return err
			}

			if size, err := io.Copy(dst, buf_reader); err != nil {
				return err
			} else {
				// update the size
				if completion != nil {
					completion(i, part.FormName(), part.FileName(), size)
				}
			}
		}
	}
	return nil
}
