package module

import (
	"fmt"
	"github.com/google/go-licenses/licenses"
	"os"
)

func (m *Module) dependencies() ([]*Module, error) {
	mods, err := m.listAll()
	if err != nil {
		return nil, err
	}
	c, err := licenses.NewClassifier(1)
	if err != nil {
		return nil, err
	}
	for _, mo := range mods {
		licenseFile, err := licenses.Find(mo.Dir, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: %s\n", err.Error())
			continue
		}
		license, lType, err := c.Identify(licenseFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: %s\n", err.Error())
			continue
		}
		fmt.Printf("module %s@%s has license %s (%s)\n", mo.Path, mo.Version, license, lType)
	}
	return nil, nil
}
