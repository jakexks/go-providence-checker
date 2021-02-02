package cmd

import (
	"fmt"
	"github.com/jakexks/go-providence-checker/pkg/checker"
	"os"
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
			license, ltype, err := s.Check(args[0])
			if err == nil {
				fmt.Printf("%s has license %s (%s)\n", args[0], license, ltype)
			}
			return err
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
			for _, l := range list {
				fmt.Println(l)
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
