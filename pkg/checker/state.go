package checker

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/jakexks/go-providence-checker/pkg/license"
)

// State holds information about the temporary go environment used
// to download and analyse transitive dependencies
type State struct {
	goPath, goCache, workingDir string
}

// Init creates new temporary directories, initialises a new go module,
// adds the package we want to use as a dependency, downloads transitive
// dependencies
func (s *State) Init(module string) error {
	log.Info("Creating Temporary Directories")
	goPath, err := newTempDir()
	if err != nil {
		return err
	}
	s.goPath = goPath
	goCache, err := newTempDir()
	if err != nil {
		return err
	}
	s.goCache = goCache
	workingDir, err := newTempDir()
	if err != nil {
		return err
	}
	s.workingDir = workingDir

	cmd := s.buildCmd("go", "mod", "init", "tmp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.Cleanup()
		log.Errorf("%s, %s", string(out), err.Error())
		return err
	}
	log.Info("Initialised empty module")
	log.Infof("Downloading %s", module)
	cmd = s.buildCmd("go", "get", module)
	out, err = cmd.CombinedOutput()
	if err != nil {
		if !viper.GetBool("force") {
			s.Cleanup()
			log.Errorf("%s, %s", string(out), err.Error())
			log.Info("Set the --force flag to continue anyway")
			os.Exit(1)
		}
	}
	cmd = s.buildCmd("go", "mod", "download")
	log.Info("Downloading transitive dependencies")
	out, err = cmd.CombinedOutput()
	if err != nil {
		s.Cleanup()
		log.Errorf("%s, %s", string(out), err.Error())
		os.Exit(1)
	}
	return nil
}

// Cleanup attempts to remove all temporary directories
func (s *State) Cleanup() {
	os.RemoveAll(s.goCache)
	os.RemoveAll(s.goPath)
	os.RemoveAll(s.workingDir)
}

// Check prints license information for a single go package
func (s *State) Check(module string) error {
	info, err := s.GoModInfo(module)
	if err != nil {
		return err
	}
	licenses, err := s.Classify(info)
	if err != nil {
		return err
	}
	log.Infof("The following licenses were found:")
	for _, li := range licenses {
		log.Infof("%s %s (%s)", li.LicenseFile, li.LicenseName, li.LicenseType)
	}
	return err
}

// Classify calls the license Classifier
func (s *State) Classify(info *GoModuleInfo) ([]license.Info, error) {
	return license.Classify(info)
}

// ListAll returns all modules used by a given package
func (s *State) ListAll() ([]*GoModuleInfo, error) {
	var allModules []*GoModuleInfo
	cmd := s.buildCmd("go", "list", "-m", "all")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	// discard first line
	buf.ReadString('\n')
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		mod := strings.Join(strings.Split(strings.TrimSpace(line), " "), "@")
		info, err := s.GoModInfo(mod)
		if err != nil {
			return nil, err
		}
		allModules = append(allModules, info)
	}
	return allModules, nil
}

// GoModInfo returns information about a go package
func (s *State) GoModInfo(module string) (*GoModuleInfo, error) {
	cmd := s.buildCmd("go", "list", "-m", "-json", module)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	info := new(GoModuleInfo)
	if err := json.Unmarshal(out, info); err != nil {
		return info, err
	}
	return info, nil
}

// buildCmd runs a command in the context of the tempoary directories from State
func (s *State) buildCmd(command string, args ...string) *exec.Cmd {
	goCmd := exec.Command(command, args...)
	goCmd.Dir = s.workingDir
	goCmd.Env = append(goCmd.Env, "GOPATH="+s.goPath, "GO111MODULE=on", "GOCACHE="+s.goCache, "PATH="+os.Getenv("PATH"))
	return goCmd
}

