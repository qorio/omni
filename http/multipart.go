package http

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type FileSink func(formName, fileName string) (io.WriteCloser, error)

var (
	MISSING_INPUT error = errors.New("missing-input")
)

func FileSystemSink(rootDir string) FileSink {
	return func(formName, fileName string) (dst io.WriteCloser, err error) {
		// Take either the form name or the filename
		id := formName
		if id == "" {
			id = fileName
		}
		if id == "" {
			return nil, MISSING_INPUT
		}
		p := filepath.Join(rootDir, id)
		dst, err = os.Create(p)
		return
	}
}

func ProcessMultiPartUpload(resp http.ResponseWriter, req *http.Request, sink FileSink) (err error) {

	//get the multipart reader for the request.
	reader, err := req.MultipartReader()

	if err != nil {
		return
	}

	//copy each part to destination.
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		dst, err := sink(part.FormName(), part.FileName())
		defer dst.Close()

		if err != nil {
			return err
		}

		if _, err := io.Copy(dst, part); err != nil {
			return err
		}
	}

	return nil
}
