package module

import (
	"github.com/spf13/viper"
	"os"
)

// cleanup is a best-effort removal of working directories, so doesn't return errors
func (m *Module) cleanup() {
	if m.initialised {
		os.RemoveAll(m.cacheDir)
		os.RemoveAll(m.workingDir)
		gopath := viper.GetString("gopath-override")
		if len(gopath) == 0 {
			os.RemoveAll(m.gopath)
		}
		m.initialised = false
	}
}
