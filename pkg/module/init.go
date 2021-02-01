package module

import (
	"github.com/spf13/viper"
	"os"
	"os/exec"
)

func (m *Module) init() error {
	if m.initialised {
		return nil
	}
	// Create temporary directories
	gopath := viper.GetString("gopath-override")
	if len(gopath) == 0 {
		tmpDir, err := newTempDir()
		if err != nil {
			return err
		}
		gopath = tmpDir
	}
	m.gopath = gopath
	workingDir, err := newTempDir()
	if err != nil {
		return err
	}
	m.workingDir = workingDir
	cacheDir, err := newTempDir()
	if err != nil {
		return err
	}
	m.cacheDir = cacheDir
	goCmd := exec.Command("go", "mod", "init", "tmp")
	goCmd.Dir = m.workingDir
	goCmd.Env = append(goCmd.Env, "GOPATH="+m.gopath, "GO111MODULE=on", "GOCACHE="+m.cacheDir, "PATH="+os.Getenv("PATH"))
	_, err = goCmd.CombinedOutput()
	m.initialised = true
	if err != nil {
		m.cleanup()
		return err
	}
	return nil
}
