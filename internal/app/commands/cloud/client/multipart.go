package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

func UploadApp(ctx context.Context, deployKey string, jar string) error {
	jarR, err := os.Open(jar)
	if err != nil {
		return fmt.Errorf("could not open file '%s': %v", jar, err)
	}
	defer jarR.Close()

	values := map[string]io.Reader{
		"token": strings.NewReader(deployKey),
		"file":  jarR,
	}

	return postMultiPart(ctx, appUploadURL, values)
}

func postMultiPart(ctx context.Context, url string, values map[string]io.Reader) error {
	var b bytes.Buffer

	w := multipart.NewWriter(&b)

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

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return err
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	return doHttpRequest(ctx, req)
}
