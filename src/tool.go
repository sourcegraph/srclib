package src

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/srclib/toolchain"
)

func toolCmd(args []string) {
	fs := flag.NewFlagSet("tool", flag.ExitOnError)
	runDirectly := fs.Bool("direct", false, "run directly (instead of in a Docker container)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` tool TOOL [ARG...]

Run a srclib tool with the specified arguments.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
	}

	toolName, toolArgs := fs.Arg(0), fs.Args()[1:]

	// if *runDirectly {
	// 	prog, err := toolchain.LookupInPATH(toolName)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	cmd := exec.Command(prog, toolArgs...)
	// 	cmd.Stdout = os.Stdout
	// 	cmd.Stderr = os.Stderr
	// 	if *Verbose {
	// 		log.Printf("Running tool %q directly: %v", toolName, cmd.Args)
	// 	}
	// 	if err := cmd.Run(); err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	return
	// }

	toolDir, err := toolchain.LookupInSRCLIBPATH(toolName)
	if err != nil {
		log.Fatal(err)
	}
	if *Verbose {
		log.Printf("Found tool %q in directory %q.", toolName, toolDir)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if !isFile(filepath.Join(toolDir, "Dockerfile")) {
		log.Fatalf("No runnable tool found in %q.", toolDir)
	}

	if *runDirectly {
		runDirectTool(dir, toolName, toolDir, toolArgs)
	} else {
		runDockerTool(dir, toolName, toolDir, toolArgs)
	}
}

func runDirectTool(dir, toolName, toolDir string, toolArgs []string) {
	data, err := ioutil.ReadFile(filepath.Join(toolDir, "Dockerfile"))
	if err != nil {
		log.Fatal(err)
	}
	var entrypointJSON []byte
	for _, line := range bytes.Split(data, []byte("\n")) {
		if bytes.HasPrefix(bytes.ToLower(line), []byte("entrypoint ")) {
			entrypointJSON = line[len("entrypoint "):]
		}
	}
	if len(entrypointJSON) == 0 {
		log.Fatalf("Dockerfile in %s must have en ENTRYPOINT instruction.", toolDir)
	}

	var cmdline []string
	if err := json.Unmarshal(entrypointJSON, &cmdline); err != nil {
		log.Fatal(err)
	}

	cmdline = append(cmdline, dir)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = toolDir
	if *Verbose {
		log.Printf("Running directly: %v", cmd.Args)
	}
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func runDockerTool(dir, toolName, toolDir string, toolArgs []string) {
	imgName := strings.Replace(toolName, "/", "-", -1)
	cmd := exec.Command("docker", "build", "-t", imgName, toolDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	// Make the Dockerfile that'll run this tool.
	// TODO(sqs): use ADD for everything but scanning?
	cmd = exec.Command("docker", "run", "--volume="+dir+":/src:ro", imgName)
	cmd.Args = append(cmd.Args, toolArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
