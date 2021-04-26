package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakexks/go-providence-checker/pkg/checker"
	"github.com/jakexks/go-providence-checker/pkg/dirutil"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	root = &cobra.Command{
		Use:   "go-providence-checker",
		Short: "ensure GCM compliance",
		Long:  `Given a go module, find all dependencies and licenses.`,
	}
	check = &cobra.Command{
		Use:   "check <module path>",
		Short: "retrieve the licence for a specific module",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s := new(checker.State)
			if err := s.Init(args[0]); err != nil {
				return fmt.Errorf("initializing go-providence-checker state: %w", err)
			}
			defer s.Cleanup()
			return s.Check(args[0])
		},
	}
	checkAll = &cobra.Command{
		Use:   "dependencies <module path>",
		Short: "retrieve the licence for a all dependencies of a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			var s checker.State
			if err := s.Init(args[0]); err != nil {
				return fmt.Errorf("while initializing go-providence-checker: %w", err)
			}
			defer s.Cleanup()

			// Cobra-specificity: runE should only return an error if this error
			// is related to the usage of the CLI. Otherwise, the error must be
			// handled and nil must be returned.
			err := run(s, args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			return nil
		},
	}
)

func init() {
	cobra.OnInitialize(flagsFromEnv)
	root.PersistentFlags().BoolP("force", "f", false, "Ignore errors during go get")
	root.AddCommand(check, checkAll)
	viper.BindPFlags(root.PersistentFlags())
}

// flagsFromEnv allows flags to be set from environment variables.
func flagsFromEnv() {
	viper.SetEnvPrefix("go_providence_checker")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func Execute() {
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// The checker state must have been already intialized with Init. The
// moduleWithTag is of the form "github.com/apache/thrift@v0.13.0".
func run(s checker.State, moduleWithTag string) error {
	list, err := s.ListAll()
	if err != nil {
		return fmt.Errorf("running checker.ListAll: %w", err)
	}
	output, err := os.OpenFile("LICENSES.txt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating LICENSES.txt: %w", err)
	}
	defer output.Close()
	seen := make(map[string]struct{})
	for _, l := range list {
		licenseInfos, err := s.Classify(l)
		switch {
		case len(licenseInfos) == 0:
			return fmt.Errorf("while classifying licenses: some licenses files were found, but the confidence level was too low")
		case err == checker.ErrNoLicenseFileFound:
			return err
		case err != nil:
			fmt.Printf("module %s@%s: no license detected, check + add manually\n", l.Path, l.Version)
		default:
			// Happy path: keep going.
		}

		for _, li := range licenseInfos {
			mod := fmt.Sprintf("%s@%s", li.LibraryName, li.LibraryVersion)
			if _, found := seen[mod]; found {
				continue
			}
			seen[mod] = struct{}{}
			fmt.Printf("module %s: %s (%s)\n", mod, li.LicenseName, li.LicenseType)
			output.Write([]byte(fmt.Sprintf("Library %s used under the %s License, reproduced below:\n\n", mod, li.LicenseName)))
			license, err := ioutil.ReadFile(li.LicenseFile)
			if err != nil {
				return fmt.Errorf("while reading license file '%s': %w", li.LicenseFile, err)
			}
			output.Write(license)
			output.Write([]byte("\n==============================\n\n"))
			switch li.LicenseType {
			case "reciprocal":
				os.MkdirAll(filepath.Join("thirdparty", li.LibraryName), 0755)
				dstPath := filepath.Join("thirdparty", li.LibraryName)
				err := dirutil.CopyDirectory(li.SourceDir, dstPath)
				if err != nil {
					return fmt.Errorf("while copying dir '%s' into '%s': %w", li.SourceDir, dstPath, err)
				}
			case "restricted":
				if !strings.HasPrefix(li.LicenseName, "LGPL") {
					return fmt.Errorf("%s is under a restricted license %s", mod, li.LicenseName)
				}

				dstPath := filepath.Join("thirdparty", li.LibraryName)

				os.MkdirAll(dstPath, 0755)
				if err := dirutil.CopyDirectory(li.SourceDir, dstPath); err != nil {
					return fmt.Errorf("while copying dir '%s' into '%s': %w", li.SourceDir, dstPath, err)
				}

				info, err := s.GoModInfo(moduleWithTag)
				if err != nil {
					return fmt.Errorf("while fetching info from the go.mod of module '%s': %w", moduleWithTag, err)
				}
				if _, found := seen["LGPL"]; !found {
					seen["LGPL"] = struct{}{}
					dstPath := filepath.Join("firstparty", info.Path)
					err = os.MkdirAll(dstPath, 0755)
					if err != nil {
						return fmt.Errorf("mkdir -p %s: %w", dstPath, err)
					}

					err = dirutil.CopyDirectory(info.Dir, filepath.Join("firstparty", info.Path))
					if err != nil {
						return fmt.Errorf("while copying dir of '%s' into '%s': %w", li.SourceDir, dstPath, err)
					}
				}

			}
		}
	}

	return nil
}
