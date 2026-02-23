package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {

	var testTools Tools

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Errorf("wrong length random string returned. expected length 10, got %d", len(s))
	}

}

var uploadTests = []struct {
	name             string
	allowedFileTypes []string
	renameFile       bool
	expectedError    bool
}{
	{name: "allowed no rename", allowedFileTypes: []string{"image/jpeg", "image/png"}, renameFile: false, expectedError: false},
	{name: "allowed rename", allowedFileTypes: []string{"image/jpeg", "image/png"}, renameFile: true, expectedError: false},
	{name: "not allowed", allowedFileTypes: []string{"image/jpeg"}, renameFile: false, expectedError: true},
}

func TestTools_UploadFile(t *testing.T) {
	for _, e := range uploadTests {
		//set up a pipe(a reader, and a writer) to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {

			defer wg.Done()
			defer pw.Close()
			defer writer.Close()

			// create the form data field
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error("error encoding image", err)
			}

		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testools Tools
		testools.AllowedFileTypes = e.allowedFileTypes

		uploadedFiles, err := testools.UploadFile(request, "./testdata/uploads/", e.renameFile)
		if err != nil {
			if e.expectedError {
				// error was expected, move on to next test case
				wg.Wait()
				continue
			}
			t.Errorf("%s: unexpected error: %s", e.name, err.Error())
			wg.Wait()
			continue
		}

		if e.expectedError {
			t.Errorf("%s: expected error but none received", e.name)
			wg.Wait()
			continue
		}

		if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
			t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
		}

		// clean up
		_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {

	//set up a pipe(a reader, and a writer) to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {

		defer pw.Close()
		defer writer.Close()

		// create the form data field
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error("error encoding image", err)
		}

	}()

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testools Tools

	uploadedFiles, err := testools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))
}
