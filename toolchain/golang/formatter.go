package golang

import (
	"encoding/json"
	"path/filepath"

	"strings"

	"sourcegraph.com/sourcegraph/srcgraph/graph"
	"sourcegraph.com/sourcegraph/srcgraph/toolchain/golang/gog"
)

func init() {
	graph.RegisterMakeSymbolFormatter(goPackageUnitType, newSymbolFormatter)
}

func newSymbolFormatter(s *graph.Symbol) graph.SymbolFormatter {
	var si SymbolData
	if err := json.Unmarshal(s.Data, &si); err != nil {
		panic("unmarshal Go symbol data: " + err.Error())
	}
	return symbolFormatter{s, &si}
}

type symbolFormatter struct {
	symbol *graph.Symbol
	info   *SymbolData
}

func (f symbolFormatter) Language() string { return "Go" }

func (f symbolFormatter) DefKeyword() string {
	switch f.info.Kind {
	case gog.Func:
		return "func"
	case gog.Var:
		if f.info.FieldOfStruct == "" && f.info.PkgScope {
			return "var"
		}
	case gog.Type:
		return "type"
	case gog.Package:
		return "package"
	}
	return ""
}

func (f symbolFormatter) Kind() string { return f.info.Kind }

func (f symbolFormatter) pkgPath(qual graph.Qualification) string {
	switch qual {
	case graph.DepQualified:
		return f.info.PkgName
	case graph.RepositoryWideQualified:
		// keep the last path component from the repo
		return strings.TrimPrefix(strings.TrimPrefix(f.info.PackageImportPath, filepath.Join(string(f.symbol.Repo), "..")), "/")
	case graph.LanguageWideQualified:
		return f.info.PackageImportPath
	}
	return ""
}

func (f symbolFormatter) Name(qual graph.Qualification) string {
	if qual == graph.Unqualified {
		return f.symbol.Name
	}

	var recvlike string
	if f.info.Kind == gog.Field {
		recvlike = f.info.FieldOfStruct
	} else if f.info.Kind == gog.Method {
		recvlike = f.info.Receiver
	}

	pkg := f.pkgPath(qual)

	if f.info.Kind == gog.Package {
		if qual == graph.ScopeQualified {
			pkg = f.symbol.Name // otherwise it'd be empty
		}
		return pkg
	}

	var prefix string
	if recvlike != "" {
		prefix = fmtReceiver(recvlike, pkg)
	} else if pkg != "" {
		prefix = pkg + "."
	}

	return prefix + f.symbol.Name
}

// fmtReceiver formats strings like `(*a/b.T).`.
func fmtReceiver(recv string, pkg string) string {
	// deref recv
	var recvName, ptrs string
	if i := strings.LastIndex(recv, "*"); i > -1 {
		ptrs = recv[:i+1]
		recvName = recv[i+1:]
	} else {
		recvName = recv
	}

	if pkg != "" {
		pkg += "."
	}

	return "(" + ptrs + pkg + recvName + ")."
}

func (f symbolFormatter) NameAndTypeSeparator() string {
	if f.info.Kind == gog.Func || f.info.Kind == gog.Method {
		return ""
	}
	return " "
}

func (f symbolFormatter) Type(qual graph.Qualification) string {
	var ts string
	switch f.symbol.Kind {
	case graph.Func:
		ts = f.info.TypeString
		ts = strings.TrimPrefix(ts, "func")
	case graph.Type:
		ts = f.info.UnderlyingTypeString
		if i := strings.Index(ts, "{"); i != -1 {
			ts = ts[:i]
		}
		ts = " " + ts
	default:
		ts = " " + f.info.TypeString
	}

	// qualify the package path based on qual
	oldPkgPath := f.info.PackageImportPath + "."
	newPkgPath := f.pkgPath(qual)
	if newPkgPath != "" {
		newPkgPath += "."
	}
	ts = strings.Replace(ts, oldPkgPath, newPkgPath, -1)
	return ts
}
