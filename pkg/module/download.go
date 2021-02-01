package module

import (
	"fmt"
	"os"
	"os/exec"
)

func (m *Module) download() (string, error) {
	if !m.initialised {
		return "", ErrModuleNotInitialised
	}
	goCmd := exec.Command("go", "get", m.Name)
	goCmd.Dir = m.workingDir
	if len(m.Version) > 0 {
		goCmd.Args[2] = goCmd.Args[2] + "@" + m.Version
	}
	goCmd.Env = append(goCmd.Env, "GOPATH="+m.gopath, "GO111MODULE=on", "GOCACHE="+m.cacheDir, "PATH="+os.Getenv("PATH"))
	out, err := goCmd.Output()
	if exitErr, ok := err.(*exec.ExitError); ok {
		fmt.Fprintf(os.Stderr, "WARN: %s: %s\n", err.Error(), string(exitErr.Stderr))
		return "", nil
	}
	return string(out), err
}
