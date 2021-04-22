package checker

import (
	"errors"
	"fmt"
	"github.com/go-enry/go-license-detector/v4/licensedb"
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
	var licenses []LicenseInfo

	result := licensedb.Analyse(info.Dir)
	if len(result) == 0 {
		s.log.Infof("no results found for %s\n", info.Dir)
	}
	for _, r := range result {
		if len(r.ErrStr) != 0 {
			s.log.Infof("FYI: %s: %s\n", info.Path, r.ErrStr)
			return s.deepClassify(info)
		}

		// Find the highest confidence license
		var max float32 = 0.0
		var license string
		var licenseFile string
		for _, m := range r.Matches {
			if m.Confidence > max {
				// Docker uses LICENSE.docs for licensing docs code
				if !strings.Contains(m.File, "docs") {
					max = m.Confidence
					license = m.License
					licenseFile = m.File
				}
			}
		}

		licenses = append(licenses, LicenseInfo{
			LibraryName:    info.Path,
			LibraryVersion: info.Version,
			LicenseFile:    filepath.Join(r.Arg, licenseFile),
			LicenseType:    licenseType(license),
			SourceDir:      info.Dir,
			LinkToLicense:  createLink(info.Path, info.Version, strings.TrimPrefix(licenseFile, info.Dir+"/")),
			LicenseName:    licenseName(license),
		})
	}

	if len(licenses) == 0 {
		return nil, ErrNoLicenseFound
	}

	return licenses, nil
}

func (s *State) deepClassify(info *GoModuleInfo) ([]LicenseInfo, error) {
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
				LicenseType:    licenseType(m.Name),
				SourceDir:      info.Dir,
				LinkToLicense:  createLink(info.Path, info.Version, strings.TrimPrefix(licenseFile, info.Dir+"/")),
				LicenseName:    licenseName(m.Name),
			})
		}
	}
	if len(licenses) == 0 {
		return nil, ErrNoLicenseFound
	}
	return licenses, nil
}

func licenseType(license string) string {
	license = licenseName(license)
	if strings.HasPrefix(license, "0BSD") {
		return "notice"
	}
	l := licenseclassifier.LicenseType(license)
	if len(l) == 0 {
		return "restricted"
	}
	return l
}

func licenseName(l string) string {
	if strings.HasPrefix(l, "MPL-2.0") {
		return "MPL-2.0"
	}
	if strings.HasPrefix(l, "LGPL-3.0") {
		return "LGPL-3.0"
	}
	if strings.HasPrefix(l, "deprecated_LGPL-3.0") {
		return "LGPL-3.0"
	}

	return l
}

func createLink(modulename, moduleversion, licensePath string) string {
	return fmt.Sprintf("https://%s/tree/%s/%s", modulename, moduleversion, licensePath)
}
