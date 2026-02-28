/*
A) initialize workspace in antigravity and create workspace in go:
- create a new folder
- file/save as workspace
- add folders to this wokrspace;
	- one folder for your module
	- one folder for your application to test your module
- run go mod init <module-name> in both folders ut be aware of naming, e.g.:
	- go mod init github.com/horvathgergo/toolkit, in case of importing this module in another module,or int the app, use this name
	- go mod init myapp - as this one is only a local app
- run go work init folder1 folder2 for both created folders
- to run a go app run go run . in the app folder
- testing go modules:
	- run go test . in the module folder
- use code repository, e.g. github:
	- create a repository using the same name as the module, check the toolkit/go.mod file for the repository name
	- run;
		- git init in the toolkit/module folder
		- git add .
		- git commit -m "initial commit"
		- follow the instruction that appeared on github after created the repository to push the code to github

B) theoretical method should always be used in case of added features:
- add functionality
- try it out
- write and run some tests
- upload to github

C) to test new features of your code sometimes you have to add new folder to both antigravity and go workspaces,
for this reason you need to:
- create a new folder
- file/add this folder to the workspace
- run go mod init <module-name> in the new folder because empty file scannot be added to go wokrspaces
- run go work use new folder

e.g. app_dir or app_upload were created to test the toolkit module this way.

slugs:
from a string create a url safe slug, e.g. "Hello World" -> "hello-world"
*/

package toolkit

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const radomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. any variable of this type will have access
// to all the methods of the receiver *Tools
type Tools struct {
	MaxFileSize        int
	AllowedFileTypes   []string
	MaxJSONSize        int
	AllowUnknownFields bool
}

// RandomString generates a random string of random characters of length n using randomStrigSource as the source of the string.
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(radomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}

	return string(s)
}

// UploadedFile is a struct used to save information about an uploaded file
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	files, err := t.UploadFile(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}

	return files[0], nil
}

func (t *Tools) UploadFile(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	for _, fHeaders := range r.MultipartForm.File {
		for _, header := range fHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				infile, err := header.Open()
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				buff := make([]byte, 512)
				_, err = infile.Read(buff)
				if err != nil {
					return nil, err
				}

				// check to see if the file is permitted
				allowed := false
				fileType := http.DetectContentType(buff)

				if len(t.AllowedFileTypes) > 0 {
					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, x) {
							allowed = true
						}
					}
				} else {
					allowed = true
				}

				if !allowed {
					return nil, errors.New("the uploaded file type is not permitted")
				}

				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(header.Filename))
				} else {
					uploadedFile.NewFileName = header.Filename
				}

				uploadedFile.OriginalFileName = header.Filename

				var outFile *os.File
				defer outFile.Close()

				if outfile, err := os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil

			}(uploadedFiles)

			if err != nil {
				return uploadedFiles, err
			}

		}
	}
	return uploadedFiles, nil
}

// CreateDirIfNotExist creates a directory and all its parent directories if they do not exist.
func (t *Tools) CreateDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

// Slugify creates a url safe slug from a string
func (t *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string is not permitted")
	}

	var re = regexp.MustCompile(`[^a-z\d]+`)
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(slug) == 0 {
		return "", errors.New("after removing characters, slug is zero length")
	}
	return slug, nil

}

// DowloadStaticFile dowloads a file, and tries to force the browser to avoid displaying it in the browser
// by setting content disposition and allows specification of the filename to be displayed in the browser
// http.ResponseWriter - we have to have somewhere to send your response
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, p, file, displayName string) {

	filePath := path.Join(p, file)
	// tells to the browser to download the file instead of opening it in the browser
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	// serves the file
	http.ServeFile(w, r, filePath)

}

type JSONResponse struct {
	Error bool   `json:"error"`
	Msg   string `json:"msg"`
	Data  any    `json:"data,omitempty"`
}

// ReadJSON tries to read the body of a request and converts from json into a go data variable
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1024 * 1024 // one megabyte
	if t.MaxJSONSize > 0 {
		maxBytes = t.MaxJSONSize
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)

	if !t.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	err := dec.Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {

		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains unexpected EOF")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body is empty")

		// for the case when the AllowUnknownFields is false
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown field %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("error unmarshalling JSON: %s", err.Error())

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

// WriteJSOM takes a response status code and arbitrary data and writes jso to client
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

// ErrorJSON takes an error and optionally a status code, and generates and sends a JSON error message
func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	payload := JSONResponse{
		Error: true,
		Msg:   err.Error(),
	}

	return t.WriteJSON(w, statusCode, payload)
}
