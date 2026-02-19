/*
0) create a new folder
1) file/save as workspace
2) add folders to this wokrspace;
	- one folder for your module
	- one folder for your application to test your module
3) run go mod init <module-name> in both folders ut be aware of naming, e.g.:
	- go mod init github.com/horvathgergo/toolkit, in case of importing this module in another module,or int the app, use this name
	- go mod init myapp - as this one is only a local app
4) run go work init folder1 folder2 for both created folders
5) to run a go app run go run . in the app folder
6) testing go modules:
	- run go test . in the module folder
7) github:
	- create a repository using the same name as the module, check the toolkit/go.mod file for the repository name
	- run;
		- git init in the toolkit/module folder
		- git add .
		- git commit -m "initial commit"
		- follow the instruction that appeared on github after created the repository to push the code to github

method:
- add functionality
- try it out
- write some tests
- upload to github

*/

package toolkit

import "crypto/rand"

const radomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. any variavle of this type will have access
// to all the methods eith the receiver *Tools
type Tools struct{}

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
