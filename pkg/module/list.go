package module

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
)

func (m *Module) list() (*GoModuleInfo, error) {
	if !m.initialised {
		return nil, ErrModuleNotInitialised
	}
	goCmd := exec.Command("go", "list", "-m", "-json", m.Name)
	goCmd.Dir = m.workingDir
	if len(m.Version) > 0 {
		goCmd.Args[4] = goCmd.Args[4] + "@" + m.Version
	}
	goCmd.Env = append(goCmd.Env, "GOPATH="+m.gopath, "GO111MODULE=on", "GOCACHE="+m.cacheDir, "PATH="+os.Getenv("PATH"))
	out, err := goCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	modInfo := &GoModuleInfo{}
	if err := json.Unmarshal(out, modInfo); err != nil {
		return nil, err
	}
	return modInfo, nil
}

func (m *Module) listAll() ([]*GoModuleInfo, error) {
	if !m.initialised {
		return nil, ErrModuleNotInitialised
	}
	goCmd := exec.Command("go", "list", "-m", "all")
	goCmd.Dir = m.workingDir
	goCmd.Env = append(goCmd.Env, "GOPATH="+m.gopath, "GO111MODULE=on", "GOCACHE="+m.cacheDir, "PATH="+os.Getenv("PATH"))
	out, err := goCmd.CombinedOutput()
	if err != nil {
		println(string(out))
		return nil, err
	}
	buf := bytes.NewBuffer(out)
	// skip first line
	buf.ReadString('\n')
	var allMods []*GoModuleInfo
	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		nameVersion := strings.Join(strings.Split(strings.TrimSpace(line), " "), "@")
		if len(strings.TrimSpace(nameVersion)) == 0 {
			continue
		}
		modInfo, err := m.mod2ModuleInfo(nameVersion)
		if err != nil {
			return nil, err
		}
		allMods = append(allMods, modInfo)
	}
	return allMods, nil
}

// GoModuleInfo represents the output of `go list`
type GoModuleInfo struct {
	Path      string `json:"Path"`
	Version   string `json:"Version"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
}

func (m *Module) mod2ModuleInfo(nameVersion string) (*GoModuleInfo, error) {
	goCmd := exec.Command("go", "list", "-m", "-json", nameVersion)
	goCmd.Dir = m.workingDir
	goCmd.Env = append(goCmd.Env, "GOPATH="+m.gopath, "GO111MODULE=on", "GOCACHE="+m.cacheDir, "PATH="+os.Getenv("PATH"))
	out, err := goCmd.CombinedOutput()
	if err != nil {
		println(string(out))
		return nil, err
	}
	modInfo := &GoModuleInfo{}
	if err := json.Unmarshal(out, modInfo); err != nil {
		return nil, err
	}
	return modInfo, nil
}
