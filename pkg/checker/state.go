package checker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	classifier "github.com/google/licenseclassifier/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type State struct {
	log                         *zap.SugaredLogger
	classifier                  *classifier.Classifier
	goPath, goCache, workingDir string
}

func (s *State) Init(module string) error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	s.log = logger.Sugar()

	s.classifier = classifier.NewClassifier(0.2)
	err = s.classifier.LoadLicenses("./licenses")
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("the folder ./licenses is missing, download it with:\n  curl -L https://github.com/google/licenseclassifier/archive/refs/tags/v2.0.0-alpha.1.tar.gz | tar xz && mv licenseclassifier-*/licenses .")
	case err != nil:
		return fmt.Errorf("loading licenses from './licenses': %w", err)
	}

	c := exec.Command("go", "env", "GOPATH")
	bytes, err := c.Output()
	if err != nil {
		return fmt.Errorf("while running 'go env GOPATH' to guess your GOPATH: %w", err)
	}
	s.goPath = strings.TrimSpace(string(bytes))

	s.log.Info("Creating Temporary Directories")
	goCache, err := newTempDir()
	if err != nil {
		return fmt.Errorf("creating temp dir for storing the temporary GOPATH: %w", err)
	}
	s.goCache = goCache
	workingDir, err := newTempDir()
	if err != nil {
		return fmt.Errorf("creating temp working dir: %w", err)
	}
	s.workingDir = workingDir

	s.log.Infof("Downloading %s", module)
	// Best effort to download modules
	cmd := s.buildCmd("go", "get", module)
	_, _ = cmd.CombinedOutput()
	modSplit := strings.Split(module, "@")
	cmd = s.buildCmd("git", "clone", "https://"+modSplit[0], "-b", modSplit[1], "./")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if !viper.GetBool("force") {
			s.Cleanup()
			s.log.Errorf("%s, %s", string(out), err.Error())
			s.log.Info("%s failed to build, use --force flag to continue anyway", module)
			os.Exit(1)
		}
	}
	cmd = s.buildCmd("go", "mod", "download")
	s.log.Info("Downloading transitive dependencies")
	out, err = cmd.CombinedOutput()
	if err != nil {
		s.Cleanup()
		s.log.Errorf("%s, %s", string(out), err.Error())
		os.Exit(1)
	}
	return nil
}

func (s *State) Cleanup() {
	os.RemoveAll(s.goCache)
	os.RemoveAll(s.workingDir)
}

func (s *State) Check(module string) error {
	info, err := s.GoModInfo(module)
	if err != nil {
		return fmt.Errorf("while reading go.mod: %w", err)
	}
	licenses, err := s.Classify(info)
	if err != nil {
		return fmt.Errorf("while classifying %v: %w", info, err)
	}
	s.log.Infof("The following licenses were found:")
	for _, li := range licenses {
		s.log.Infof("%s %s (%s)", li.LicenseFile, li.LicenseName, li.LicenseType)
	}
	return nil
}

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
		var mod string
		if strings.Contains(line, "=>") {
			replacedMod := strings.Split(line, " => ")[1]
			mod = strings.Join(strings.Split(strings.TrimSpace(replacedMod), " "), "@")
		} else {
			mod = strings.Join(strings.Split(strings.TrimSpace(line), " "), "@")
		}
		info, err := s.GoModInfo(mod)
		if err != nil {
			return nil, err
		}
		allModules = append(allModules, info)
	}
	return allModules, nil
}

func (s *State) GoModInfo(module string) (*GoModuleInfo, error) {
	cmd := s.buildCmd("go", "list", "-m", "-json", module)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s failed\n", module)
		return nil, err
	}

	info := GoModuleInfo{}
	if err := json.Unmarshal(out, &info); err != nil {
		return &info, err
	}
	return &info, nil
}
