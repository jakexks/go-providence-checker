package checker

import (
	"bytes"
	"encoding/json"
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
		if viper.GetBool("force") {
			return nil
		}
		s.Cleanup()
		s.log.Errorf("%s, %s", string(out), err.Error())
		s.log.Info("Set the --force flag to continue anyway")
		os.Exit(1)
		return nil
	}
	return nil
}

func (s *State) Cleanup() {
	os.RemoveAll(s.goCache)
	os.RemoveAll(s.goPath)
	os.RemoveAll(s.workingDir)
}

func (s *State) Check(module string) error {
	cmd := s.buildCmd("go", "list", "-m", "-json", module)
	s.log.Info("Parsing module " + module)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	info := new(GoModuleInfo)
	if err := json.Unmarshal(out, info); err != nil {
		return err
	}
	license, lType, err := s.classify(info)
	if err != nil {
		if err == ErrNoLicenseFound {
			license = "none"
			lType = licenses.Forbidden
		} else {
			return err
		}
	}
	s.log.Infof("Module %s has license %s (%s)", module, license, lType)
	return nil
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
		if err := s.Check(mod); err != nil {
			s.log.Error(err)
		}
	}
	return output, nil
}
