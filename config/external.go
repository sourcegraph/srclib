package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/srclib/toolchain"
)

// DefaultScanners are the scanners used for a Tree if none are manually
// specified in a Srcfile.
var DefaultScanners = []*toolchain.ToolRef{
	{"github.com/sourcegraph/srclib-go", "scan"},
}

// An External configuration file, represented by this struct, can set system-
// and user-level settings for srclib.
type External struct {
	// DefaultScanners is the default set of scanners to use.
	DefaultScanners []*toolchain.ToolRef
}

// SrclibPathConfig is stored in SRCLIBPATH/.srclibconfig.
var SrclibPathConfig External

const srclibconfigFile = ".srclibconfig"

func init() {
	dir := strings.SplitN(toolchain.SrclibPath, ":", 2)[0]
	configFile := filepath.Join(dir, srclibconfigFile)
	f, err := os.Open(configFile)
	if os.IsNotExist(err) {
		// do nothing
	} else if err != nil {
		log.Printf("Warning: unable to open config file at %s: %s. Continuing without this config.", configFile, err)
	} else {
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&SrclibPathConfig); err != nil {
			log.Printf("Warning: unable to decode config file at %s: %s. Continuing without this config.", configFile, err)
		}
	}

	// Use default scanners.
	if len(SrclibPathConfig.DefaultScanners) == 0 {
		SrclibPathConfig.DefaultScanners = DefaultScanners
	}
}
