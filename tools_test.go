package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
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

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testools Tools
	err := testools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	err = testools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	// clean up
	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	toSlugify     string
	expected      string
	errorExpected bool
}{
	{name: "valid string", toSlugify: "Now is the Time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", toSlugify: "", expected: "", errorExpected: true},
	{name: "japanese string", toSlugify: "こんにちは世界", expected: "", errorExpected: true},
	{name: "japanese and roman string", toSlugify: "Hello こんにちは世界 world", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testtools Tools

	for _, e := range slugTests {
		slug, err := testtools.Slugify(e.toSlugify)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: unexpected error: %s", e.name, err.Error())
		}

		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: expected slug %s, but got %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testtool Tools
	testtool.DownloadStaticFile(rr, req, "./testdata", "img.png", "downloadedAndRenamed.png")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "534283" {
		t.Errorf("expected content length of 534283, got %s", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"downloadedAndRenamed.png\"" {
		t.Errorf("wrong content disposition, expected: ttachment; filename=\"downloadedAndRenamed.png\", got %s", res.Header["Content-Disposition"][0])
	}

}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "valid json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "invalid json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": 1}{"bar": 2}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{"foo": 1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field in json", json: `{"fooo": "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown field in json", json: `{"fooo": "1"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 1, allowUnknown: true},
	{name: "not json", json: `hello, world!`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, e := range jsonTests {
		// set max file size
		testTool.MaxJSONSize = e.maxSize
		// set allow/disallow unknown fields
		testTool.AllowUnknownFields = e.allowUnknown

		// declare a variable to read the decoded json into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a request with the body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		// create a recorder
		rr := httptest.NewRecorder()

		// call the ReadJSON method
		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: expected error but not received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: unexpected error: %s", e.name, err.Error())
		}

		req.Body.Close()

	}

}

func TestTools_WriteJSON(t *testing.T) {

	var testTool Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error: false,
		Msg:   "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTool.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}

}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding JSON", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}

}
