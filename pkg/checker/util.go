package checker

import (
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alessio/shellescape"
)

func newTempDir() (string, error) {
	tmpDir := os.TempDir() + "/go-providence-checker-" + strconv.Itoa(rand.Int())
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", err
	}
	return tmpDir, nil
}

func (s *State) buildCmd(cmd string, args ...string) *exec.Cmd {
	goCmd := exec.Command(cmd, args...)
	goCmd.Dir = s.workingDir
	// The GOCACHE is required because this command is run without a HOME
	// env. See: https://github.com/golang/go/issues/29267
	goCmd.Env = append(goCmd.Env, "GO111MODULE=on", "GOCACHE="+s.goCache, "GOPATH="+s.goPath, "PATH="+os.Getenv("PATH"))
	s.Log.Debugf(PrettyCommand(cmd, args...))
	return goCmd
}

// PrettyCommand takes arguments identical to Cmder.Command,
// it returns a pretty printed command that could be pasted into a shell
func PrettyCommand(name string, args ...string) string {
	var out strings.Builder
	out.WriteString(shellescape.Quote(name))
	for _, arg := range args {
		out.WriteByte(' ')
		out.WriteString(shellescape.Quote(arg))
	}
	return out.String()
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
