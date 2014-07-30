//+build off

package src

import (
	"flag"
	"fmt"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srclib/unit"
	"sourcegraph.com/sourcegraph/srclib/vcsutil"
)

func blame(args []string) {
	fs := flag.NewFlagSet("blame", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` blame [options] [unit...]

Blames the files in a source unit. If no source units are specified, all source
units are blamed.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)
	sourceUnitSpecs := fs.Args()

	repoConf, err := OpenAndConfigureRepo(Dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range repoConf.Config.SourceUnits {
		if !SourceUnitMatchesArgs(sourceUnitSpecs, u) {
			continue
		}

		files, err := unit.ExpandPaths(repoConf.RootDir, u.Files)
		if err != nil {
			log.Fatal(err)
		}

		var out *vcsutil.BlameOutput
		if files == nil {
			out, err = vcsutil.BlameRepository(repoConf.RootDir, repoConf.CommitID, repoConf.Config)
		} else {
			out, err = vcsutil.BlameFiles(repoConf.RootDir, files, repoConf.CommitID, repoConf.Config)
		}
		if err != nil {
			log.Fatal(err)
		}
		PrintJSON(out, "")
	}
}
