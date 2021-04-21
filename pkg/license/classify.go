package license

import (
	"errors"
	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/kr/pretty"
	"os"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"

	"github.com/jakexks/go-providence-checker/pkg/checker"
)

var (
	possibleLicense   = regexp.MustCompile(`^(?i)(LICEN(S|C)E|COPYING|README|NOTICE)(\\..+)?$`)
	ErrNoLicenseFound = errors.New("no license found")
)


func Classify(info *checker.GoModuleInfo) ([]Info, error) {
	licenseFiles, err := findObviousLicenses(info)
	if err != nil {
		return nil, err
	}
	if len(licenseFiles) == 0 {
		log.Info("no licenses found, attempting fallback")
		return nil, errors.New("fallback not implemented")
	} else {
		results := licensedb.Analyse(licenseFiles...)
		log.Infof("%# v\n", pretty.Formatter(results))
	}

	return nil, errors.New("not implemented")
}

func findObviousLicenses(info *checker.GoModuleInfo) ([]string, error) {
	// Walk directory for obvious license files
	var licenseFiles []string
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
		return licenseFiles, err
	}
	return licenseFiles, nil
}