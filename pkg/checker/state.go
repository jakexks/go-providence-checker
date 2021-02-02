package checker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/go-licenses/licenses"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"io"
	"os"
	"strings"
)

type State struct {
	log                         *zap.SugaredLogger
	classifier                  licenses.Classifier
	goPath, goCache, workingDir string
}

func (s *State) Init(module string) error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	s.log = logger.Sugar()

	c, err := licenses.NewClassifier(1)
	if err != nil {
		return err
	}
	s.classifier = c

	s.log.Info("Creating Temporary Directories")
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
		s.log.Errorf("%s, %s", string(out), err.Error())
		return err
	}
	s.log.Info("Initialised empty module")
	s.log.Infof("Downloading %s", module)
	cmd = s.buildCmd("go", "get", module)
	out, err = cmd.CombinedOutput()
	if err != nil {
		if !viper.GetBool("force") {
			s.Cleanup()
			s.log.Errorf("%s, %s", string(out), err.Error())
			s.log.Info("Set the --force flag to continue anyway")
			os.Exit(1)
		}
	}
	cmd = s.buildCmd("go", "mod", "download")
	s.log.Info("Downloading dependencies")
	out, err = cmd.CombinedOutput()
	if err != nil {
		s.Cleanup()
		s.log.Errorf("%s, %s", string(out), err.Error())
		os.Exit(1)
	}
	return nil
}

func (s *State) Cleanup() {
	//os.RemoveAll(s.goCache)
	//os.RemoveAll(s.goPath)
	//os.RemoveAll(s.workingDir)
}

func (s *State) Check(module string) (string, licenses.Type, error) {
	cmd := s.buildCmd("go", "list", "-m", "-json", module)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", err
	}
	info := new(GoModuleInfo)
	if err := json.Unmarshal(out, info); err != nil {
		return "", "", err
	}
	return s.classify(info)
}

func (s *State) ListAll() ([]string, error) {
	var output []string
	cmd := s.buildCmd("go", "list", "-m", "all")
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		return output, err
	}
	// discard first line
	buf.ReadString('\n')
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return output, err
		}
		mod := strings.Join(strings.Split(strings.TrimSpace(line), " "), "@")
		if license, ltype, err := s.Check(mod); err != nil {
			s.log.Infof("%s: %s", mod, err)
		} else {
			output = append(output, fmt.Sprintf("%s %s (%s)", mod, license, ltype))
		}
	}
	return output, nil
}
