package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sourcegraph/srclib/toolchain"
)

// An External configuration file, represented by this struct, can set system-
// and user-level settings for srclib.
type External struct {
	// Scanners is the default set of scanners to use. If not specified, all
	// scanners in the SRCLIBPATH will be used.
	Scanners []*toolchain.ToolRef
}

// SrclibPathConfig is stored in SRCLIBPATH/.srclibconfig.
var SrclibPathConfig External

const srclibconfigFile = ".srclibconfig"

func init() {
	dir := strings.SplitN(srclib.Path, ":", 2)[0]
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

	// Default to using all available scanners.
	if len(SrclibPathConfig.Scanners) == 0 {
		SrclibPathConfig.Scanners, err = toolchain.ListTools("scan")
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("Failed to find scanners in SRCLIBPATH: %s.", err)
		}
	}
}
