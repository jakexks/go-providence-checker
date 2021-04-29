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
	root.PersistentFlags().BoolP("debug", "d", false, "Print commands being that are run in the background")
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
// rootMod is of the form "github.com/apache/thrift@v0.13.0".
func run(s checker.State, rootMod string) error {
	gomodEntries, err := s.GoList()
	if err != nil {
		return fmt.Errorf("running checker.ListAll: %w", err)
	}
	licensestxt, err := os.OpenFile("LICENSES.txt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating LICENSES.txt: %w", err)
	}
	defer licensestxt.Close()
	seen := make(map[string]struct{})
	for _, entry := range gomodEntries {
		li, err := s.Classify(entry)
		switch {
		case err == checker.ErrNoLicenseFileFound:
			if viper.GetBool("force") {
				s.Log.Infof("module %s@%s: no license file found in the directory '%s'", entry.Path, entry.Version, entry.Dir)
				continue
			} else {
				return fmt.Errorf("module %s@%s: no license file found in the directory '%s'. Run with --force to ignore.", entry.Path, entry.Version, entry.Dir)
			}
		case err != nil:
			fmt.Printf("module %s@%s: no license detected, check + add manually\n", entry.Path, entry.Version)
			continue
		default:
			// Happy path: keep going.
		}

		mod := fmt.Sprintf("%s@%s", li.LibraryName, li.LibraryVersion)

		if _, found := seen[mod]; found {
			continue
		}
		seen[mod] = struct{}{}

		fmt.Printf("module %s: %s (%s)\n", mod, li.LicenseName, li.LicenseType)

		_, err = licensestxt.Write([]byte(fmt.Sprintf("Library %s used under the %s License, reproduced below:\n\n", mod, li.LicenseName)))
		if err != nil {
			return fmt.Errorf("module %s: while writing to LICENSES.txt: %w", mod, err)
		}

		license, err := ioutil.ReadFile(li.LicenseFile)
		if err != nil {
			s.Log.Debugf("%s: go mod entry is %#v", mod, li)
			return fmt.Errorf("module %s: while reading license file '%s': %w", mod, li.LicenseFile, err)
		}
		licensestxt.Write(license)
		licensestxt.Write([]byte("\n==============================\n\n"))
		switch li.LicenseType {
		case "reciprocal":
			dstPath := filepath.Join("thirdparty", li.LibraryName)
			os.MkdirAll(dstPath, 0755)
			err := dirutil.CopyDirectory(li.SourceDir, dstPath)
			if err != nil {
				return fmt.Errorf("while copying the source code for the dependency '%s' due to the reprocical license %s, copying '%s' into '%s': %w", mod, li.LicenseName, li.SourceDir, dstPath, err)
			}
		case "restricted":
			if !strings.HasPrefix(li.LicenseName, "LGPL") {
				if viper.GetBool("force") {
					s.Log.Infof("module %s: the license %s is restricted but is not LGPL, cannot continue. Run with --force to ignore.", mod, li.LicenseName)
					continue
				} else {
					return fmt.Errorf("module %s: the license %s is restricted but is not LGPL, cannot continue. Run with --force to ignore.", mod, li.LicenseName)
				}
			}

			dstPath := filepath.Join("thirdparty", li.LibraryName)

			os.MkdirAll(dstPath, 0755)
			err := dirutil.CopyDirectory(li.SourceDir, dstPath)
			if err != nil {
				return fmt.Errorf("while copying the root's source code for the dependency '%s' due to its restricted license %s: copying dir '%s' into '%s': %w", mod, li.LicenseName, li.SourceDir, dstPath, err)
			}

			// We need to copy the source code of the module given by the
			// user, since this restricted dependency requires source code
			// to be distributed.
			info, err := s.GoListSingle(rootMod)
			if err != nil {
				return fmt.Errorf("while copying the root's source code for the dependency '%s' due to its restricted license %s: while go listing '%s': %w", mod, li.LicenseName, mod, err)
			}

			if _, found := seen["LGPL"]; found {
				continue
			}
			seen["LGPL"] = struct{}{}

			dstPath = filepath.Join("firstparty", info.Path)

			err = os.MkdirAll(dstPath, 0755)
			if err != nil {
				return fmt.Errorf("mkdir -p %s: %w", dstPath, err)
			}

			err = dirutil.CopyDirectory(info.Dir, dstPath)
			if err != nil {
				return fmt.Errorf("while copying the root's source code for the dependency '%s' due to its restricted license %s: while copying dir '%s' into '%s': %w", mod, li.LicenseName, li.SourceDir, dstPath, err)
			}
		}
	}

	return nil
}
