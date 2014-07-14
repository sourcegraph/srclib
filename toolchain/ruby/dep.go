package ruby

import (
	"encoding/json"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/config"
	"sourcegraph.com/sourcegraph/srcgraph/container"
	"sourcegraph.com/sourcegraph/srcgraph/dep2"
	"sourcegraph.com/sourcegraph/srcgraph/unit"
)

func init() {
	dep2.RegisterLister(&RubyGem{}, dep2.DockerLister{DefaultRubyVersion})
	dep2.RegisterResolver(rubygemTargetType, DefaultRubyVersion)
}

func (v *Ruby) BuildLister(dir string, unit unit.SourceUnit, c *config.Repository) (*container.Command, error) {
	rubygem := unit.(*RubyGem)

	dockerfile, err := v.baseDockerfile()
	if err != nil {
		return nil, err
	}

	containerDir := "/tmp/rubygem"
	cont := container.Container{
		Dockerfile: dockerfile,
		AddDirs:    [][2]string{{dir, containerDir}},
		Dir:        filepath.Join(containerDir, rubygem.RootDir()),
		Cmd:        []string{"rvm", "all", "do", "ruby", "-rbundler", "-rjson", "-e", `puts JSON.generate(Hash[Bundler.definition.dependencies.map{|d|[d.name, d.requirement.to_s]}]) if File.exist?("Gemfile")`},
	}

	cmd := container.Command{
		Container: cont,
		Transform: func(orig []byte) ([]byte, error) {
			if len(orig) == 0 {
				// no Gemfile
				return []byte("[]"), nil
			}

			var gemDeps map[string]string
			err := json.Unmarshal(orig, &gemDeps)
			if err != nil {
				return nil, err
			}

			var deps []*dep2.RawDependency
			for gemName, version := range gemDeps {
				if gemName == rubygem.GemName {
					// Skip the gem we're analyzing.
					continue
				}
				deps = append(deps, &dep2.RawDependency{
					TargetType: rubygemTargetType,
					Target: &rubyGemDep{
						Name:    gemName,
						Version: version,
					},
				})
			}

			return json.Marshal(deps)
		},
	}
	return &cmd, nil
}

// rubyGemDep represents a RubyGem dependency.
type rubyGemDep struct {
	Name    string `json:",omitempty"`
	Version string `json:",omitempty"`
}

const rubygemTargetType = "rubygem"

func (v *Ruby) Resolve(dep *dep2.RawDependency, c *config.Repository) (*dep2.ResolvedTarget, error) {
	// Remarshal dep.Target so we can unmarshal it as a *rubyGemDep.
	tmpJSON, err := json.Marshal(dep.Target)
	if err != nil {
		return nil, err
	}
	var gemDep *rubyGemDep
	err = json.Unmarshal(tmpJSON, &gemDep)
	if err != nil {
		return nil, err
	}

	return v.resolveRubyGemDep(gemDep, c)
}

func (v *Ruby) resolveRubyGemDep(gemDep *rubyGemDep, c *config.Repository) (*dep2.ResolvedTarget, error) {
	gemName := gemDep.Name

	// Look up in cache.
	resolvedTarget := func() *dep2.ResolvedTarget {
		v.resolveCacheMu.Lock()
		defer v.resolveCacheMu.Unlock()
		return v.resolveCache[gemName]
	}()
	if resolvedTarget != nil {
		return resolvedTarget, nil
	}

	resolvedTarget = &dep2.ResolvedTarget{
		ToUnit:          gemDep.Name,
		ToUnitType:      rubygemUnitType,
		ToVersionString: gemDep.Version,
	}

	// Look it up on rubygems.org.
	repoURL, err := ResolveGem(gemDep.Name)
	if err != nil {
		log.Printf("Failed to resolve RubyGem dependency %v: %s (continuing)", gemDep, err)
		return nil, nil
	}
	resolvedTarget.ToRepoCloneURL = repoURL

	// Save in cache.
	v.resolveCacheMu.Lock()
	defer v.resolveCacheMu.Unlock()
	if v.resolveCache == nil {
		v.resolveCache = make(map[string]*dep2.ResolvedTarget)
	}
	v.resolveCache[gemName] = resolvedTarget

	return resolvedTarget, nil
}

// isGitHubRepoCloneURL determines if urlStr is a GitHub repo URL: if the host
// is github.com and path is "/user/repo" (with 2 slashes).
func isGitHubRepoCloneURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Host == "github.com" && strings.Count(filepath.Clean(u.Path), "/") == 2
}
