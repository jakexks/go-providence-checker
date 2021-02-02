package checker

import (
	"errors"
	"github.com/google/go-licenses/licenses"
	"os"
	"path/filepath"
)

var ErrNoLicenseFound = errors.New("no license found")

func (s *State) classify(info *GoModuleInfo) (string, licenses.Type, error) {
	licenseFile, err := licenses.Find(info.Dir, s.classifier)
	if err != nil {
		s.log.Infof("Trying %s", filepath.Join(info.Dir, "LICENSE"))
		if f, err := os.Open(filepath.Join(info.Dir, "LICENSE")); err == nil {
			f.Close()
			return s.classifier.Identify(filepath.Join(info.Dir, "LICENSE"))
		}
		s.log.Infof("Trying %s", filepath.Join(info.Dir, "LICENCE"))
		if f, err := os.Open(filepath.Join(info.Dir, "LICENCE")); err == nil {
			f.Close()
			return s.classifier.Identify(filepath.Join(info.Dir, "LICENCE"))
		}
		return "", "", err
	}
	return s.classifier.Identify(licenseFile)
}
