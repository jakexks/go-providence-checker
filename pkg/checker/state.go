package checker

import (
	"bytes"
	"encoding/json"
	classifier "github.com/google/licenseclassifier/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"io"
	"os"
	"strings"
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
	if err := s.classifier.LoadLicenses("./licenses"); err != nil {
		return err
	}

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
	os.RemoveAll(s.goPath)
	os.RemoveAll(s.workingDir)
}

func (s *State) Check(module string) error {
	info, err := s.GoModInfo(module)
	if err != nil {
		return err
	}
	licenses, err := s.Classify(info)
	if err != nil {
		return err
	}
	s.log.Infof("The following licenses were found:")
	for _, li := range licenses {
		s.log.Infof("%s %s (%s)", li.LicenseFile, li.LicenseName, li.LicenseType)
	}
	return err
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
		mod := strings.Join(strings.Split(strings.TrimSpace(line), " "), "@")
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
		return nil, err
	}
	info := new(GoModuleInfo)
	if err := json.Unmarshal(out, info); err != nil {
		return info, err
	}
	return info, nil
}
