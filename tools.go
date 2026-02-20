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
- run go mod init <module-name> in the new folder
- run go work use new folder
*/

package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const radomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. any variavle of this type will have access
// to all the methods eith the receiver *Tools
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
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

	err := r.ParseMultipartForm(int64(t.MaxFileSize))
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
