package http

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type PartCheck func(part int, formName, fileName string, peek io.Reader) error
type PartSink func(part int, formName, fileName string) (io.WriteCloser, error)

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

func ProcessMultiPartUpload(resp http.ResponseWriter, req *http.Request, check PartCheck, sink PartSink) (err error) {

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

		if check != nil {
			err := check(i, part.FormName(), part.FileName(), part)
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

			if _, err := io.Copy(dst, part); err != nil {
				return err
			}
		}
	}
	return nil
}
