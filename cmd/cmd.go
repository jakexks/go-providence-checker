package cmd

import (
	"fmt"
	"github.com/jakexks/go-providence-checker/pkg/checker"
	"github.com/jakexks/go-providence-checker/pkg/dirutil"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			s := new(checker.State)
			if err := s.Init(args[0]); err != nil {
				return err
			}
			defer s.Cleanup()
			return s.Check(args[0])
		},
	}
	checkAll = &cobra.Command{
		Use:   "dependencies <module path>",
		Short: "retrieve the licence for a all dependencies of a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := new(checker.State)
			if err := s.Init(args[0]); err != nil {
				return err
			}
			defer s.Cleanup()
			list, err := s.ListAll()
			if err != nil {
				fmt.Printf("listAll failed\n")
				return err
			}
			output, err := os.OpenFile("LICENSES.txt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer output.Close()
			seen := make(map[string]struct{})
			for _, l := range list {
				licenseInfo, err := s.Classify(l)
				if err != nil {
					if err == checker.ErrNoLicenseFound {
						fmt.Printf("\nmodule %s@%s: no license detected, check + add manually\n\n", l.Path, l.Version)
					} else {
						return err
					}
				}
				for _, li := range licenseInfo {
					mod := fmt.Sprintf("%s@%s", li.LibraryName, li.LibraryVersion)
					if _, found := seen[mod]; found {
						continue
					}
					seen[mod] = struct{}{}
					fmt.Printf("module %s: %s (%s)\n", mod, li.LicenseName, li.LicenseType)
					output.Write([]byte(fmt.Sprintf("Library %s used under the %s License, reproduced below:\n\n", mod, li.LicenseName)))
					license, err := ioutil.ReadFile(li.LicenseFile)
					if err != nil {
						return err
					}
					output.Write(license)
					output.Write([]byte("\n==============================\n\n"))
					switch li.LicenseType {
					case "reciprocal":
						os.MkdirAll(filepath.Join("thirdparty", li.LibraryName), 0755)
						if err := dirutil.CopyDirectory(li.SourceDir, filepath.Join("thirdparty", li.LibraryName)); err != nil {
							return err
						}
					case "restricted":
						if strings.HasPrefix(li.LicenseName, "LGPL") {
							os.MkdirAll(filepath.Join("thirdparty", li.LibraryName), 0755)
							if err := dirutil.CopyDirectory(li.SourceDir, filepath.Join("thirdparty", li.LibraryName)); err != nil {
								return err
							}
							info, err := s.GoModInfo(args[0])
							if err != nil {
								return err
							}
							if _, found := seen["LGPL"]; !found {
								seen["LGPL"] = struct{}{}
								os.MkdirAll(filepath.Join("firstparty", info.Path), 0755)
								if err := dirutil.CopyDirectory(info.Dir, filepath.Join("firstparty", info.Path)); err != nil {
									return err
								}
							}
						} else {
							return fmt.Errorf("%s is under a restricted license %s", mod, li.LicenseName)
						}
					}
				}
			}
			return err
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
