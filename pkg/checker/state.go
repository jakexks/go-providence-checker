package checker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	classifier "github.com/google/licenseclassifier/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type State struct {
	Log                         *zap.SugaredLogger
	classifier                  *classifier.Classifier
	goPath, goCache, workingDir string
}

func (s *State) Init(module string) error {
	var opts []zap.Option
	if viper.GetBool("debug") {
		opts = append(opts, zap.IncreaseLevel(zap.DebugLevel))
	}

	logger, err := zap.NewDevelopment(opts...)
	if err != nil {
		return err
	}
	s.Log = logger.Sugar()

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

	s.Log.Info("Creating Temporary Directories")
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

	s.Log.Infof("Downloading %s", module)
	// Best effort to download modules
	cmd := s.buildCmd("go", "get", module)
	_, _ = cmd.CombinedOutput()
	modSplit := strings.Split(module, "@")

	// When the module that is given to go-providence-checker has "replace"
	// directives in its go.mod, such as:
	//
	//  go-providence-checker dependencies github.com/jetstack/preflight@v0.1.29
	//
	// we can't just "go get" since "go get" ignores the replace directives. To
	// work around that, the only way is to actually clone the repo. The
	// guesswork we do to find the HTTPS URL may not always work since we mainly
	// wanted it to work for publicly hosted GitHub repos.
	cmd = s.buildCmd("git", "clone", "https://"+modSplit[0], "-b", modSplit[1], "./")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if !viper.GetBool("force") {
			s.Cleanup()
			s.Log.Errorf("%s, %s", string(out), err.Error())
			os.Exit(1)
		}
	}
	cmd = s.buildCmd("go", "mod", "download")
	s.Log.Info("Downloading transitive dependencies")
	out, err = cmd.CombinedOutput()
	if err != nil {
		s.Cleanup()
		s.Log.Errorf("module %s: command 'go mod download' in directory '%s': %s.\nThe stderr/stdout were:\n%s\n. Use --force to ignore.", module, workingDir, string(out), err)
		os.Exit(1)
	}
	return nil
}

func (s *State) Cleanup() {
	os.RemoveAll(s.goCache)
	os.RemoveAll(s.workingDir)
}

func (s *State) Check(module string) error {
	info, err := s.GoListSingle(module)
	if err != nil {
		return fmt.Errorf("while reading go.mod: %w", err)
	}

	li, err := s.Classify(info)
	if err != nil {
		return fmt.Errorf("while classifying %v: %w", info, err)
	}
	s.Log.Infof("The following license was found:")
	s.Log.Infof("%s %s (%s)", li.LicenseFile, li.LicenseName, li.LicenseType)
	return nil
}

func (s *State) GoListSingle(module string) (GoModuleInfo, error) {
	modules, err := s.GoList(module)
	if err != nil {
		return GoModuleInfo{}, err
	}
	if len(modules) != 1 {
		return GoModuleInfo{}, fmt.Errorf("programmer mistake: Check: a single module was expected to be returned")
	}
	return modules[0], nil
}

// When no module is given, all the modules will be listed.
func (s *State) GoList(modules ...string) ([]GoModuleInfo, error) {
	args := []string{"list", "-m", "-json"}
	args = append(args, modules...)
	if len(modules) == 0 {
		args = append(args, "all")
	}
	cmd := s.buildCmd("go", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("while running 'go %v': %w", args, err)
	}

	return parseGoListJsonOutput(out)
}

// The go list -json command does not return an actual array of json
// objects. Instead, it "streams" the json objects. This solution is highly
// inspired from:
// https://github.com/golang/go/issues/27655#issuecomment-420993215.
func parseGoListJsonOutput(b []byte) ([]GoModuleInfo, error) {
	var modules []GoModuleInfo
	dec := json.NewDecoder(bytes.NewReader(b))
	for {
		var m GoModuleInfo
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("reading 'go list -json' output: %v", err)
		}

		modules = append(modules, m)
	}

	return modules, nil
}
