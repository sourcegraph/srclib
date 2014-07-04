package main

import (
	"flag"
	"log"
	"os"

	"sourcegraph.com/sourcegraph/srcgraph/grapher/ruby"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("Usage: sg-gem-repo GEM\n\nwhere GEM is a Ruby gem on rubygems.org")
	}

	gemName := flag.Arg(0)
	log.Printf("Resolving Ruby gem %q using hard-coded list or rubygems.org API...", gemName)

	cloneURL, gemPath, err := ruby.ResolveGem(gemName)
	if err != nil {
		log.Fatalf("Failed to resolve gem: %s", err)
	}
	log.Printf("\nResolved gem %q:\n\n", gemName)
	log.Printf("Clone URL: %s", cloneURL)
	log.Printf("Gem path (relative to repository root): %s", gemPath+"/")
}
