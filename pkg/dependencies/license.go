package dependencies

import (
	"fmt"
	"github.com/jakexks/go-providence-checker/pkg/module"
)

// License gets the license for a single module
func License(module *module.Module) error {
	if err := module.Init(); err != nil {
		return err
	}
	defer module.Cleanup()
	if err := module.Download(); err != nil {
		return err
	}
	license, lType, err := module.License()
	if err != nil {
		return err
	}
	if len(module.Version) > 0 {
		fmt.Printf("module %s@%s has license %s (%s)\n", module.Name, module.Version, license, lType)
	} else {
		fmt.Printf("module %s has license %s (%s)\n", module.Name, license, lType)
	}
	return nil
}
