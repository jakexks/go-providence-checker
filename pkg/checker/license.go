package checker

import (
	"errors"
	"fmt"
	"github.com/google/licenseclassifier"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	possibleLicense   = regexp.MustCompile(`^(?i)(LICEN(S|C)E|COPYING|README|NOTICE)(\\..+)?$`)
	ErrNoLicenseFound = errors.New("no license found")
)

type LicenseInfo struct {
	LibraryName    string
	LibraryVersion string
	LicenseFile    string
	SourceDir      string
	LinkToLicense  string
	LicenseName    string
	LicenseType    string
}

func (s *State) Classify(info *GoModuleInfo) ([]LicenseInfo, error) {
	var licenseFiles []string
	var licenses []LicenseInfo
	if err := filepath.Walk(info.Dir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			if possibleLicense.MatchString(fileInfo.Name()) {
				licenseFiles = append(licenseFiles, path)
			}
		}
		return nil
	}); err != nil {
		return licenses, err
	}
	for _, licenseFile := range licenseFiles {
		content, err := ioutil.ReadFile(licenseFile)
		if err != nil {
			return licenses, err
		}
		matches := s.classifier.Match(content)
		for _, m := range matches {
			licenses = append(licenses, LicenseInfo{
				LibraryName:    info.Path,
				LibraryVersion: info.Version,
				LicenseFile:    licenseFile,
				LicenseType:    licenseclassifier.LicenseType(m.Name),
				SourceDir:      info.Dir,
				LinkToLicense:  createLink(info.Path, info.Version, strings.TrimPrefix(licenseFile, info.Dir+"/")),
				LicenseName:    m.Name,
			})
		}
	}
	if len(licenses) == 0 {
		return nil, ErrNoLicenseFound
	}
	return licenses, nil
}

func createLink(modulename, moduleversion, licensePath string) string {
	return fmt.Sprintf("https://%s/tree/%s/%s", modulename, moduleversion, licensePath)
}
