package module

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/go-licenses/licenses"
)

type Module struct {
	// Name is the module name, e.g. github.com/jetstack/cert-manager
	Name string
	// Version is the module version, e.g. v1.0.0
	Version string

	initialised bool
	gopath      string
	workingDir  string
	cacheDir    string
	env         []string
}

var ErrModuleNotInitialised = errors.New("module not initialised")

// Init creates a working area for the module downloads
func (m *Module) Init() error {
	return m.init()
}

// Download downloads the module to disk
func (m *Module) Download() error {
	out, err := m.download()
	if err != nil {
		fmt.Fprintln(os.Stderr, out)
		return err
	}
	return nil
}

// Cleanup deletes the working area from disk
func (m *Module) Cleanup() {
	m.cleanup()
}

// License gets the module License
func (m *Module) License() (string, licenses.Type, error) {
	return m.license()
}

// Dependencies gets a list of dependencies and their Licenses
func (m *Module) Dependencies() ([]*Module, error) {
	return m.dependencies()
}
