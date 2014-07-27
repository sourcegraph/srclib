package srclib

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

var (
	// Path is SRCLIBPATH, a colon-separated list of directories that lists
	// places to look for srclib toolchains and cache build data. It is
	// initialized from the SRCLIBPATH environment variable; if empty, it
	// defaults to $HOME/.srclib.
	Path = os.Getenv("SRCLIBPATH")

	// CacheDir stores cached build results. It is initialized from the
	// SRCLIBCACHE environment variable; if empty, it defaults to DIR/.cache,
	// where DIR is the first entry in Path (SRCLIBPATH).
	CacheDir = os.Getenv("SRCLIBCACHE")
)

func init() {
	if Path == "" {
		user, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		if user.HomeDir == "" {
			log.Fatalf("Fatal: No SRCLIBPATH and current user %q has no home directory.", user.Username)
		}
		Path = filepath.Join(user.HomeDir, ".srclib")
		if err := os.Setenv("SRCLIBPATH", Path); err != nil {
			log.Fatalf("Fatal: Could not set SRCLIBPATH environment variable to %q.", Path)
		}
	}

	if CacheDir == "" {
		dirs := strings.SplitN(Path, ":", 2)
		CacheDir = filepath.Join(dirs[0], ".cache")
	}
}
