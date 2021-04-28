package checker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/google/licenseclassifier"
	classifier "github.com/google/licenseclassifier/v2"
)

var (
	licenseFileRegex      = regexp.MustCompile(`^(?i)(LICEN(S|C)E|COPYING|README|NOTICE)(\\..+)?$`)
	ErrNoLicenseFileFound = errors.New("not able to find a license file in this directory")
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

func (s *State) Classify(info GoModuleInfo) (LicenseInfo, error) {
	license, err := fastClassify(info)
	if err == nil {
		return license, nil
	}
	if err != ErrNoLicenseFileFound {
		return LicenseInfo{}, err
	}

	s.Log.Infof("%s: go-license-detector didn't find anything, falling back to google/licenseclassifier", info.Path)

	license, err = deepClassify(s.classifier, info)
	if err == nil {
		return license, nil
	}
	if err != ErrNoLicenseFileFound {
		return LicenseInfo{}, err
	}

	return LicenseInfo{}, ErrNoLicenseFileFound
}

// Returns ErrNoLicenseFileFound when no license can be found in the
// module's tree.
func fastClassify(info GoModuleInfo) (LicenseInfo, error) {
	// A single result is returned since we give a single directory.
	results := licensedb.Analyse(info.Dir)
	if len(results) == 0 {
		return LicenseInfo{}, errors.New("developer mistake since one result = one dir")
	}
	result := results[0]
	matches := result.Matches

	switch result.ErrStr {
	case "no license file was found":
		return LicenseInfo{}, ErrNoLicenseFileFound
	case "":
		// No error, let's continue.
	default:
		return LicenseInfo{}, fmt.Errorf("using go-license-detector: %s", result.ErrStr)
	}

	if len(matches) == 0 {
		return LicenseInfo{}, ErrNoLicenseFileFound
	}

	var candidates []candidate
	for _, match := range matches {
		candidates = append(candidates, candidate{
			license:    match.License,
			confidence: float64(match.Confidence),
			path:       filepath.Join(result.Arg, match.File),
		})
	}

	highest := highestConfidence(candidates)

	return LicenseInfo{
		LibraryName:    info.Path,
		LibraryVersion: info.Version,
		LicenseFile:    highest.path,
		LicenseType:    licenseType(highest.license),
		SourceDir:      info.Dir,
		LinkToLicense:  createLink(info.Path, info.Version, strings.TrimPrefix(highest.path, info.Dir+"/")),
		LicenseName:    licenseName(highest.license),
	}, nil
}

type candidate struct {
	path       string  // Absolute path to the license file.
	license    string  // Of the form "BSD-3-Clause".
	confidence float64 // Some relative number, the higher the more confident.
}

func highestConfidence(c []candidate) candidate {
	highest := candidate{confidence: -math.MaxInt64}
	for _, current := range c {
		if current.confidence < highest.confidence {
			continue
		}

		// Docker uses LICENSE.docs for licensing docs code, we don't care
		// about documentation licenses.
		if strings.HasSuffix(current.path, "LICENSE.docs") {
			continue
		}

		highest = current
	}

	return highest
}

// Use Google's slow licenseclassifier to find the possible licenses in the
// whole project's directory tree.
func deepClassify(c *classifier.Classifier, info GoModuleInfo) (LicenseInfo, error) {
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
		return LicenseInfo{}, fmt.Errorf("walking the tree starting at '%s': %w", info.Dir, err)
	}

	var candidates []candidate
	for _, licenseFile := range licenseFiles {
		content, err := ioutil.ReadFile(licenseFile)
		if err != nil {
			return LicenseInfo{}, fmt.Errorf("reading license file '%s': %w", licenseFile, err)
		}
		for _, m := range c.Match(content) {
			candidates = append(candidates, candidate{
				license:    m.Name,
				confidence: m.Confidence,
				path:       licenseFile,
			})
		}
	}

	if len(candidates) == 0 {
		return LicenseInfo{}, ErrNoLicenseFileFound
	}

	highest := highestConfidence(candidates)

	return LicenseInfo{
		LibraryName:    info.Path,
		LibraryVersion: info.Version,
		LicenseFile:    highest.path,
		LicenseType:    licenseType(highest.license),
		SourceDir:      info.Dir,
		LinkToLicense:  createLink(info.Path, info.Version, strings.TrimPrefix(highest.path, info.Dir+"/")),
		LicenseName:    licenseName(highest.license),
	}, nil
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
