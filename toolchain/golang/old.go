// +build off

package golang

import (
	"fmt"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/scan/unit"
)

// esc shell-escapes a string when it is interpolated into a Go template.
type esc string

func (s esc) String() string { return fmt.Sprintf("%q", s) }

func (t *goToolchain) containerBaseRepoDir(config *config.Repository) string {
	return filepath.Join(containerGOPATH, "src", config.Go.BaseImportPath)
}

func (t *goToolchain) RepositoryDockerfile(repoCloneDir string, config *config.Repository) (string, error) {
	// Set up a symlink so that the Go packages in this repository have their
	// import paths in this env's GOPATH be prefixed with the BaseImportPath.
	// This ensures that importing other packages in this repository imports the
	// dependency package at the same revision of the package we're working on.
	baseRepoDir := t.containerBaseRepoDir(config)
	return t.Dockerfile() + `
RUN mkdir -p ` + shellescape(filepath.Dir(baseRepoDir)) + `
RUN ln -s ` + shellescape(repoCloneDir) + ` ` + shellescape(baseRepoDir) + `
WORKDIR ` + shellescape(baseRepoDir) + "\n"
}

func (t *goToolchain) SourceUnitDockerfile(unit unit.SourceUnit, repoCloneDir string, config *config.Repository) (string, error) {
	baseRepoDir := t.containerBaseRepoDir(config)
	return t.RepositoryDockerfile(repoCloneDir, config) + `
WORKDIR ` + shellescape(baserepoDir) + "\n"
}

// goRepository represents the repository containing a Go source unit. When
// processing a Go package, the rest of the repository must also be added to the
// GOPATH to ensure that imports of other packages in the same repository
// resolve to the versions in the same commit as the source unit. If the imports
// were resolved using `go get`, then they would always resolve to the
// master/tip branch.
//
// The goRepository and RepositoryCheckout components are redundant. If you use
// goRepository, you shouldn't have a RepositoryCheckout.
type goRepository struct {
	//	toolchain      *goBuildContext
	baseImportPath string
}

func (r *goRepository) Description() string {
	return fmt.Sprintf("Copy Go repository %s to GOPATH", r.baseImportPath)
}

func (r *goRepository) buildSteps() (string, error) {
	return fmt.Sprintf(`# Add the entire repository for this Go package to the $GOPATH
RUN mkdir -p %q
ADD . %s
WORKDIR %s`, filepath.Dir(baseImportPathDir), baseImportPathDir, baseImportPathDir), nil
}

// goPackage represents a Go package that will be added to an environment.
type goPackage struct {
	//	toolchain                *goBuildContext
	repositoryBaseImportPath string
	pkg                      *unit.GoPackage
}

func (p *goPackage) Description() string {
	return fmt.Sprintf("Install dependencies for Go package %s", p.pkg.ImportPath)
}

// buildSteps installs dependencies of this package.
func (p *goPackage) buildSteps() (string, error) {
	importPaths, err := p.pkg.DependencyImportPaths(p.toolchain.buildContext)
	if err != nil {
		return "", err
	}

	var instructs []string
	for _, ip := range importPaths {
		if _, isBuiltin := p.toolchain.Stdlib.BuiltinPackages[ip]; isBuiltin {
			continue
		}

		// Packages from the same repository are added to the GOPATH by
		// `type goRepository` above. Don't fetch them with `go get` or we will
		// reset to the wrong version.
		if ip == p.repositoryBaseImportPath || strings.HasPrefix(ip, p.repositoryBaseImportPath+"/") {
			continue
		}

		cmd, err := p.GoGet(ip)
		if err != nil {
			return "", err
		}
		instructs = append(instructs, fmt.Sprintf("RUN %s", strings.Join(cmd, " ")))
	}
	if len(instructs) == 0 {
		instructs = append(instructs, "# (no external package imports)")
	}
	return strings.Join(instructs, "\n"), nil
}

func (p *goPackage) GoGet(importPath string) ([]string, error) {
	return p.goBuildCmd("get", []string{"-d"}, importPath)
}

func (p *goPackage) GoInstall(importPath string) ([]string, error) {
	return p.goBuildCmd("install", nil, importPath)
}

func (p *goPackage) goBuildCmd(cmdName string, flags []string, importPath string) ([]string, error) {
	cmd := []string{p.toolchain.goCmd, cmdName}
	cmd = append(cmd, flags...)

	// Construct the -tags list (according to the format described in `go help
	// build`).
	if len(p.toolchain.buildContext.BuildTags) > 0 {
		cmd = append(cmd, "-tags", goListFlagValue(p.toolchain.buildContext.BuildTags))
	}

	cmd = append(cmd, "--", importPath)
	return cmd, nil
}

// goListFlagValue constructs a list flag value according to the specification
// in `go help build`:
//
// The list flags accept a space-separated list of strings. To embed spaces
// in an element in the list, surround it with either single or double quotes.
func goListFlagValue(list []string) string {
	for i, e := range list {
		// TODO(sqs): This fails for build tags with double quotes in them. But
		// that's probably very rare.
		list[i] = `"` + e + `"`
	}
	return strings.Join(list, " ")
}

type goGrapherCommand struct {
	//	toolchain *goBuildContext
	pkg *unit.GoPackage
}

func (c *goGrapherCommand) Description() string {
	return fmt.Sprintf("Graph Go package %s", c.pkg.ImportPath)
}

func (c *goGrapherCommand) buildSteps() (string, error) {
	// TODO(sqs): add build tags
	return fmt.Sprintf(`CMD /sg/bin/sg-golang-grapher %s`, c.pkg.ImportPath), nil
}
