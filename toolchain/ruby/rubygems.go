package ruby

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sync"
)

// GemCloneURLs maps rubygems.org gem names to clone URLs (or resolution
// errors). It is updated by ResolveGem as gems are resolved, but hardcoded
// entries will never be overwritten (and override rubygems.org). Errors are
// cached so that we don't hit rubygems.org continually if resolution fails.
var GemCloneURLs = map[string]gemResolution{
	"ruby": {"git://github.com/ruby/ruby", nil},

	"rails":         {"git://github.com/rails/rails", nil},
	"actionmailer":  {"git://github.com/rails/rails", nil},
	"actionpack":    {"git://github.com/rails/rails", nil},
	"actionview":    {"git://github.com/rails/rails", nil},
	"activerecord":  {"git://github.com/rails/rails", nil},
	"activemodel":   {"git://github.com/rails/rails", nil},
	"railties":      {"git://github.com/rails/rails", nil},
	"activesupport": {"git://github.com/rails/rails", nil},

	"elasticsearch":            {"git://github.com/elasticsearch/elasticsearch-ruby", nil},
	"elasticsearch-api":        {"git://github.com/elasticsearch/elasticsearch-ruby", nil},
	"elasticsearch-extensions": {"git://github.com/elasticsearch/elasticsearch-ruby", nil},
	"elasticsearch-transport":  {"git://github.com/elasticsearch/elasticsearch-ruby", nil},

	"sass":                      {"git://github.com/nex3/sass", nil},
	"json":                      {"git://github.com/flori/json", nil},
	"treetop":                   {"git://github.com/nathansobo/treetop", nil},
	"barkick":                   {"git://github.com/ankane/barkick", nil},
	"groupdate":                 {"git://github.com/ankane/groupdate", nil},
	"pretender":                 {"git://github.com/ankane/pretender", nil},
	"searchkick":                {"git://github.com/ankane/searchkick", nil},
	"chartkick":                 {"git://github.com/ankane/chartkick", nil},
	"redis":                     {"git://github.com/redis/redis-rb", nil},
	"geocoder":                  {"git://github.com/alexreisner/geocoder", nil},
	"yajl":                      {"git://github.com/brianmario/yajl-ruby", nil},
	"plu":                       {"git://github.com/ankane/plu", nil},
	"active_median":             {"git://github.com/ankane/active_median", nil},
	"delayed_job":               {"git://github.com/collectiveidea/delayed_job", nil},
	"delayed_job_active_record": {"git://github.com/collectiveidea/delayed_job_active_record", nil},
	"tire-contrib":              {"git://github.com/karmi/tire-contrib", nil},
}

type gemResolution struct {
	CloneURL string
	err      error
}

var resolveGemLock sync.Mutex

var cleanRE = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

var (
	ErrNoGemFound     = errors.New("gem not found on rubygems.org")
	ErrGemInvalidName = errors.New("gem name contains invalid characters")
	ErrGemEmptyName   = errors.New("gem name is empty")
	ErrGemNoRepoURL   = errors.New("gem has no repository URL")
)

func ResolveGem(name string) (cloneURL string, err error) {
	cleanName := cleanRE.ReplaceAllLiteralString(name, "")
	if name != cleanName {
		return "", ErrGemInvalidName
	}
	if name == "" {
		return "", ErrGemEmptyName
	}

	c, present := GemCloneURLs[name]
	if present {
		return c.CloneURL, c.err
	}

	defer func() {
		GemCloneURLs[name] = gemResolution{cloneURL, err}
	}()

	var resp *http.Response
	resp, err = http.Get("https://rubygems.org/api/v1/gems/" + name + ".json")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ErrNoGemFound
	}

	type rubyGem struct {
		SourceCodeURI string `json:"source_code_uri"`
		HomepageURI   string `json:"homepage_uri"`
	}
	var info *rubyGem
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return
	}

	if info.SourceCodeURI != "" {
		return info.SourceCodeURI, nil
	}
	if isGitHubRepoCloneURL(info.HomepageURI) {
		return info.HomepageURI, nil
	}

	return "", ErrGemNoRepoURL
}
