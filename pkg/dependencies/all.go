package dependencies

import "github.com/jakexks/go-providence-checker/pkg/module"

// All gets all dependencies of a module
func All(module *module.Module) error {
	if err := module.Init(); err != nil {
		return err
	}
	defer module.Cleanup()
	if err := module.Download(); err != nil {
		return err
	}
	deps, err := module.Dependencies()
	if err != nil {
		return err
	}
	for _ = range deps {

	}
	return nil
}
