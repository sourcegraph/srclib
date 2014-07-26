package toolchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// A ToolRef identifies a tool inside a specific toolchain. It can be used to
// look up the tool.
type ToolRef struct {
	// Toolchain is the toolchain path of the toolchain that contains this tool.
	Toolchain string

	// Subcmd is the name of the toolchain subcommand that runs this tool.
	Subcmd string
}

func (t ToolRef) String() string { return fmt.Sprintf("%s %s", t.Toolchain, t.Subcmd) }

func (t *ToolRef) UnmarshalFlag(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return errors.New("expected format 'TOOLCHAIN:TOOL' (separated by 1 colon)")
	}
	t.Toolchain = parts[0]
	t.Subcmd = parts[1]
	return nil
}

func (t ToolRef) MarshalFlag() (string, error) {
	return t.Toolchain + ":" + t.Subcmd, nil
}

// ToolInfo describes a tool in a toolchain.
type ToolInfo struct {
	// Toolchain is the toolchain that contains this tool as a subcommand.
	Toolchain *Info

	// Subcmd is the subcommand name of this tool.
	Subcmd string

	// Op is the operation that this tool performs.
	Op string
}

// Ref returns the ToolRef that refers to this tool.
func (t *ToolInfo) Ref() *ToolRef { return &ToolRef{t.Toolchain.Path, t.Subcmd} }

// ListTools lists all tools in all available toolchains (returned by List). If
// op is non-empty, only tools that perform that operation are returned.
func ListTools(op string) ([]*ToolInfo, error) {
	tcs, err := List()
	if err != nil {
		return nil, err
	}

	var tools []*ToolInfo
	for _, tc := range tcs {
		tcTools, err := tc.Tools()
		if err != nil {
			return nil, err
		}

		for _, tool := range tcTools {
			if op == "" || tool.Op == op {
				tools = append(tools, tool)
			}
		}
	}
	return tools, nil
}

// OpenTool opens a tool in toolchain (which is a toolchain path) named subcmd.
// The mode parameter controls how the toolchain is opened.
func OpenTool(toolchain, subcmd string, mode Mode) (Tool, error) {
	tc, err := Open(toolchain, mode)
	if err != nil {
		return nil, err
	}

	return &tool{tc, subcmd}, nil
}

// A Tool is a subcommand of a Toolchain that performs an single operation, such
// as one type of analysis on a source unit.
type Tool interface {
	// Command returns an *exec.Cmd suitable for running this tool.
	Command() (*exec.Cmd, error)

	// Run executes this tool with args and parses the JSON response into resp.
	Run(arg []string, resp interface{}) error
}

type tool struct {
	tc     Toolchain
	subcmd string
}

func (t *tool) Command() (*exec.Cmd, error) {
	// make command
	cmd, err := t.tc.Command()
	if err != nil {
		return nil, err
	}
	cmd.Args = append(cmd.Args, t.subcmd)
	return cmd, nil
}

func (t *tool) Run(arg []string, resp interface{}) error {
	cmd, err := t.Command()
	if err != nil {
		return err
	}
	cmd.Args = append(cmd.Args, arg...)

	log.Printf("Run: %v", cmd.Args)

	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := json.NewDecoder(stdout).Decode(resp); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}
