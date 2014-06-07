package python

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
)

func init() {
	graph.RegisterMakeSymbolFormatter(DistPackageDisplayName, newSymbolFormatter)
}

func newSymbolFormatter(s *graph.Symbol) graph.SymbolFormatter {
	var si symbolData
	if len(s.Data) > 0 {
		if err := json.Unmarshal(s.Data, &si); err != nil {
			panic("unmarshal Python symbol data: " + err.Error())
		}
	}
	return symbolFormatter{s, &si}
}

type symbolFormatter struct {
	symbol *graph.Symbol
	data   *symbolData
}

func (f symbolFormatter) Language() string { return "Python" }

func (f symbolFormatter) DefKeyword() string {
	if f.isFunc() {
		return "def"
	}
	if f.data.Kind == "class" {
		return "class"
	}
	if f.data.Kind == "module" {
		return "module"
	}
	if f.data.Kind == "package" {
		return "package"
	}
	return ""
}

func (f symbolFormatter) Kind() string { return f.data.Kind }

func dotted(slashed string) string { return strings.Replace(slashed, "/", ".", -1) }

func (f symbolFormatter) Name(qual graph.Qualification) string {
	if qual == graph.Unqualified {
		return f.symbol.Name
	}

	// Get the name of the containing package or module
	var containerName string
	if filename := filepath.Base(f.symbol.File); filename == "__init__.py" {
		containerName = filepath.Base(filepath.Dir(f.symbol.File))
	} else if strings.HasSuffix(filename, ".py") {
		containerName = filename[:len(filename)-len(".py")]
	} else {
		// Should never reach here, but fall back to TreePath if we do
		return string(f.symbol.TreePath)
	}

	// Compute the path relative to the containing package or module
	var treePathCmps = strings.Split(string(f.symbol.TreePath), "/")
	// Note(kludge): The first occurrence of the container name in the treepath may not be the correct occurrence.
	containerCmpIdx := -1
	for t, component := range treePathCmps {
		if component == containerName {
			containerCmpIdx = t
			break
		}
	}
	var relTreePath string
	if containerCmpIdx != -1 {
		relTreePath = strings.Join(treePathCmps[containerCmpIdx+1:], "/")
		if relTreePath == "" {
			relTreePath = "."
		}
	} else {
		// Should never reach here, but fall back to the unqualified name if we do
		relTreePath = f.symbol.Name
	}

	switch qual {
	case graph.ScopeQualified:
		return dotted(relTreePath)
	case graph.DepQualified:
		return dotted(filepath.Join(containerName, relTreePath))
	case graph.RepositoryWideQualified:
		return dotted(string(f.symbol.TreePath))
	case graph.LanguageWideQualified:
		return string(f.symbol.Repo) + "/" + f.Name(graph.RepositoryWideQualified)
	}
	panic("Name: unhandled qual " + string(qual))
}

func (f symbolFormatter) isFunc() bool {
	k := f.data.Kind
	return k == "function" || k == "method" || k == "constructor"
}

func (f symbolFormatter) NameAndTypeSeparator() string {
	if f.isFunc() {
		return ""
	}
	return " "
}

func (f symbolFormatter) Type(qual graph.Qualification) string {
	fullSig := f.data.FuncSignature
	if strings.Contains(fullSig, ")") { // kludge to get rid of extra type info (very noisy)
		return fullSig[:strings.Index(fullSig, ")")+1]
	}
	return fullSig
}
