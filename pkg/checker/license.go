package checker

import (
	"errors"
	"github.com/google/go-licenses/licenses"
)

var ErrNoLicenseFound = errors.New("no license found")

func (s *State) classify(info *GoModuleInfo) (string, licenses.Type, error) {
	licenseFile, err := licenses.Find(info.Dir, s.classifier)
	if err != nil {
		return "", "", ErrNoLicenseFound
	}
	return s.classifier.Identify(licenseFile)
}
