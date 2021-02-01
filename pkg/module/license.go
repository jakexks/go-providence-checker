package module

import (
	"errors"
	"github.com/google/go-licenses/licenses"
)

func (m *Module) license() (string, licenses.Type, error) {
	if !m.initialised {
		return "", "", ErrModuleNotInitialised
	}
	modinfo, err := m.list()
	if err != nil {
		return "", "", err
	}
	c, err := licenses.NewClassifier(1)
	if err != nil {
		return "", "", err
	}
	licenseFile, err := licenses.Find(modinfo.Dir, c)
	if err != nil {
		return "", "", errors.New("no license found")
	}
	return c.Identify(licenseFile)
}
