package client

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"cli-enonic/internal/app/commands/cloud/util"
)

type UploadAppResponse struct {
	Data JarInfo `json:"data"`
}

type JarInfo struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Icon     string `json:"icon"`
	IconType string `json:"iconType"`
}

// UploadApp uses deploy key to upload an app to the Cloud API
func UploadApp(ctx context.Context, solutionID string, jar string, pbMessage string) (string, error) {
	// Open jar
	jarR, err := os.Open(jar)
	if err != nil {
		return "", fmt.Errorf("could not open file '%s': %v", jar, err)
	}
	defer jarR.Close()

	// Get Jar info
	fi, err := jarR.Stat()
	if err != nil {
		return "", err
	}

	// Create multipart body
	values := map[string]io.Reader{
		"solutionId": strings.NewReader(solutionID),
		"file":       jarR,
	}
	contentType, reader, processFunc := createMultipartBody(values)

	extraBytes := 300 // This is set to the progress bar ends at 100%

	pb := util.CreateProgressBar(fi.Size()+int64(len(solutionID)+extraBytes), pbMessage)
	pb.Start()
	defer pb.Finish()

	// Do request
	res := new(UploadAppResponse)
	err = postMultiPart(ctx, appUploadURL, contentType, pb.NewProxyReader(reader), processFunc, &res)
	if err != nil {
		return "", err
	}

	return res.Data.ID, nil
}

// Create a multipart body for http request. This function returns the content type header, byte reader and a function to
// start sending bytes to the reader
func createMultipartBody(values map[string]io.Reader) (string, io.Reader, func() error) {
	// Setup multipart request
	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)

	processFunc := func() error {
		// Create body
		for key, r := range values {
			var fw io.Writer
			var err error
			if x, ok := r.(io.Closer); ok {
				defer x.Close()
			}

			if x, ok := r.(*os.File); ok {
				if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
					return err
				}
			} else {
				if fw, err = w.CreateFormField(key); err != nil {
					return err
				}
			}

			if _, err := io.Copy(fw, r); err != nil {
				return err
			}
		}
		w.Close()
		pw.Close()
		return nil
	}

	return w.FormDataContentType(), pr, processFunc
}

// This function executes a multipart request
func postMultiPart(ctx context.Context, url string, contentType string, reader io.Reader, processFunc func() error, res interface{}) error {
	// Create request
	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

	var pError error
	go func() {
		pError = processFunc()
	}()

	// Do request
	err = doHTTPRequest(ctx, req, res)

	// Return errors if present
	if err != nil {
		return err
	}
	if pError != nil {
		return pError
	}
	return nil
}
