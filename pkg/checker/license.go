package checker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/google/licenseclassifier"
)

var (
	licenseFileRegex      = regexp.MustCompile(`^(?i)(LICEN(S|C)E|COPYING|README|NOTICE)(\\..+)?$`)
	ErrNoLicenseFileFound = errors.New("the license detector (github.com/go-enry/go-license-detector) was not able to find a license in the directory")
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

// Returns ErrNoLicenseFileDetected when no license can be found in the module's
// tree. May return an empty list of license infos even when license files are
// found, because the level of confidence is not high enough for those license
// files.
func (s *State) Classify(info *GoModuleInfo) ([]LicenseInfo, error) {
	var licenses []LicenseInfo
	results := licensedb.Analyse(info.Dir)
	if len(results) == 0 {
		return nil, ErrNoLicenseFileFound
	}
	for _, result := range results {
		if len(result.ErrStr) != 0 {
			s.Log.Infof("FYI %s: %s; now trying to find licenses inside Go files using Google's classifier", info.Path, result.ErrStr)
			licences, err := s.deepClassify(info)
			if err != nil {
				return nil, fmt.Errorf("using Google's classifier: %w", err)
			}

			return licences, nil
		}

		// Find the highest confidence license.
		var highest float32 = 0.0
		var highestLicense string
		var highestFile string
		for _, m := range result.Matches {
			if m.Confidence < highest {
				continue
			}

			// Docker uses LICENSE.docs for licensing docs code, we don't care
			// about documentation licenses.
			if strings.Contains(m.File, "docs") {
				continue
			}

			highest = m.Confidence
			highestLicense = m.License
			highestFile = m.File
		}

		licenses = append(licenses, LicenseInfo{
			LibraryName:    info.Path,
			LibraryVersion: info.Version,
			LicenseFile:    filepath.Join(result.Arg, highestFile),
			LicenseType:    licenseType(highestLicense),
			SourceDir:      info.Dir,
			LinkToLicense:  createLink(info.Path, info.Version, strings.TrimPrefix(highestFile, info.Dir+"/")),
			LicenseName:    licenseName(highestLicense),
		})
	}

	return licenses, nil
}

// May return an empty list of license infos even when license files are found,
// because the level of confidence is not high enough for those license files.
func (s *State) deepClassify(info *GoModuleInfo) ([]LicenseInfo, error) {
	var licenseFiles []string
	err := filepath.Walk(info.Dir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walking the tree starting at '%s': %w", info.Dir, err)
		}
		if fileInfo.IsDir() {
			return nil
		}
		if !licenseFileRegex.MatchString(fileInfo.Name()) {
			return nil
		}

		licenseFiles = append(licenseFiles, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking the tree starting at '%s': %w", info.Dir, err)
	}

	var licenses []LicenseInfo
	for _, licenseFile := range licenseFiles {
		content, err := ioutil.ReadFile(licenseFile)
		if err != nil {
			return nil, fmt.Errorf("reading license file '%s': %w", licenseFile, err)
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
