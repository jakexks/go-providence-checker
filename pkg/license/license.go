// package license takes a go module and returns information about contained licenses
package license

type Info struct {
	// PackageName is the go import path e.g. github.com/jetstack/cert-manager
	PackageName string
	// PackageVersion is the version or go pseudo-version e.g. v1.0.0
	PackageVersion string
	// LicenseFile is the path to the LICENSE file on Disk, e.g. /tmp/mod/cache/lib@version/LICENSE
	LicenseFile string
	// SourceDir is the location of the go source for the package on disk e.g. /tmp/mod/cache/lib@version/
	SourceDir string
	// LinkToLicense is a link to the LICENSE on the web, e.g. https://github.com/jetstack/cert-manager/tree/v1.0.0/LICENSE
	LinkToLicense string
	// LicenseName is the SPDX license name
	LicenseName string
	// LicenseType is the license category as defined by https://pkg.go.dev/github.com/google/licenseclassifier#LicenseType
	LicenseType string
}
