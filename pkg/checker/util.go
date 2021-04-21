package checker

import (
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func newTempDir() (string, error) {
	tmpDir := os.TempDir() + "/go-providence-checker-" + strconv.Itoa(rand.Int())
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", err
	}
	return tmpDir, nil
}

func (s *State) buildCmd(command string, args ...string) *exec.Cmd {
	goCmd := exec.Command(command, args...)
	goCmd.Dir = s.workingDir
	goCmd.Env = append(goCmd.Env, "GOPATH="+s.goPath, "GO111MODULE=on", "GOCACHE="+s.goCache, "PATH="+os.Getenv("PATH"))
	return goCmd
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type GoModuleInfo struct {
	Path      string `json:"Path"`
	Version   string `json:"Version"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
}
