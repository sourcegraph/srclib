package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"

	"strings"
	"unicode"

	"text/scanner"

	"sourcegraph.com/sourcegraph/go-flags"

	"sourcegraph.com/sourcegraph/srclib/config"
	"sourcegraph.com/sourcegraph/srclib/cvg"
	"sourcegraph.com/sourcegraph/srclib/graph"
	"sourcegraph.com/sourcegraph/srclib/grapher"
	"sourcegraph.com/sourcegraph/srclib/plan"
	"sourcegraph.com/sourcegraph/srclib/unit"
)

func init() {
	cliInit = append(cliInit, func(cli *flags.Command) {
		_, err := cli.AddCommand("coverage",
			"srclib coverage",
			"compute approximate amount of code successfully analyzed by srclib",
			&coverageCmd,
		)
		if err != nil {
			log.Fatal(err)
		}
	})
}

type codeFileDatum struct {
	LoC          int
	NumRefs      int
	NumDefs      int
	NumRefsValid int
}

type CoverageCmd struct {
}

var coverageCmd CoverageCmd

func (c *CoverageCmd) Execute(args []string) error {
	repo, err := OpenLocalRepo()
	if err != nil {
		return err
	}

	cvg, err := coverage(repo)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(cvg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))

	return nil
}

var codeExts = []string{".go", ".java", ".py", ".rb", ".cpp", ".ts", ".cs", ".js", ".php", ".m"} // codeExt lists all file extensions that indicate a code file we want to cover
var codeExts_ = make(map[string]struct{})

func init() {
	for _, ext := range codeExts {
		codeExts_[ext] = struct{}{}
	}
}

func coverage(repo *Repo) (*cvg.Coverage, error) {
	codeFileData := make(map[string]*codeFileDatum)
	log.Printf(repo.RootDir)
	filepath.Walk(repo.RootDir, func(path string, info os.FileInfo, err error) error {
		if filepath.IsAbs(path) {
			var err error
			path, err = filepath.Rel(repo.RootDir, path)
			if err != nil {
				return err
			}
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir // don't search hidden directories
			}
			return nil
		}

		path = filepath.ToSlash(path)

		if _, isCodeFile := codeExts_[filepath.Ext(path)]; isCodeFile {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			loc := numLines(b)
			codeFileData[path] = &codeFileDatum{LoC: loc}
		}
		return nil
	})

	bdfs, err := GetBuildDataFS(repo.CommitID)
	if err != nil {
		return nil, err
	}

	treeConfig, err := config.ReadCached(bdfs)
	if err != nil {
		return nil, fmt.Errorf("error calling config.ReadCached: %s", err)
	}
	mf, err := plan.CreateMakefile(".", nil, "", treeConfig)
	if err != nil {
		return nil, fmt.Errorf("error calling plan.Makefile: %s", err)
	}

	defKeys := make(map[graph.DefKey]struct{})
	data := make([]graph.Output, 0, len(mf.Rules))

	parseGraphData := func(graphFile string, sourceUnit *unit.SourceUnit) error {
		var item graph.Output
		if err := readJSONFileFS(bdfs, graphFile, &item); err != nil {
			if err == errEmptyJSONFile {
				log.Printf("Warning: the JSON file is empty for unit %s %s.", sourceUnit.Type, sourceUnit.Name)
				return nil
			}
			if os.IsNotExist(err) {
				log.Printf("Warning: no build data for unit %s %s.", sourceUnit.Type, sourceUnit.Name)
				return nil
			}
			return fmt.Errorf("error reading JSON file %s for unit %s %s: %s", graphFile, sourceUnit.Type, sourceUnit.Name, err)
		}
		data = append(data, item)

		for _, def := range item.Defs {
			defKeys[adjustDefKey(def.DefKey, sourceUnit)] = struct{}{}
		}

		for _, ref := range item.Refs {
			ref.SetFromDefKey(adjustDefKey(ref.DefKey(), sourceUnit))
		}
		return nil
	}

	for _, rule_ := range mf.Rules {
		switch rule := rule_.(type) {
		case *grapher.GraphUnitRule:
			if err := parseGraphData(rule.Target(), rule.Unit); err != nil {
				return nil, err
			}
		case *grapher.GraphMultiUnitsRule:
			for target, sourceUnit := range rule.Targets() {
				if err := parseGraphData(target, sourceUnit); err != nil {
					return nil, err
				}
			}
		}
	}

	for _, item := range data {

		var validRefs []*graph.Ref
		for _, ref := range item.Refs {
			if datum, exists := codeFileData[ref.File]; exists {
				datum.NumRefs++

				if ref.DefRepo != "" {
					validRefs = append(validRefs, ref)
					datum.NumRefsValid++
				} else if _, defExists := defKeys[ref.DefKey()]; defExists {
					validRefs = append(validRefs, ref)
					datum.NumRefsValid++
				}
			}
		}

		for _, def := range item.Defs {
			if datum, exists := codeFileData[def.File]; exists {
				datum.NumDefs++
			}
		}
	}

	var fileTokThresh float64 = 0.7
	numIndexedFiles := 0
	numDefs, numRefs, numRefsValid := 0, 0, 0
	loc := 0 // lines of code
	var uncoveredFiles []string
	for file, datum := range codeFileData {
		loc += datum.LoC
		numDefs += datum.NumDefs
		numRefs += datum.NumRefs
		numRefsValid += datum.NumRefsValid
		if float64(datum.NumDefs+datum.NumRefsValid)/float64(datum.LoC) > fileTokThresh {
			numIndexedFiles++
		} else {
			uncoveredFiles = append(uncoveredFiles, file)
		}
	}

	return &cvg.Coverage{
		FileScore:      divideSentinel(float64(numIndexedFiles), float64(len(codeFileData)), -1),
		RefScore:       divideSentinel(float64(numRefsValid), float64(numRefs), -1),
		TokDensity:     divideSentinel(float64(numDefs+numRefs), float64(loc), -1),
		UncoveredFiles: uncoveredFiles,
	}, nil
}

func divideSentinel(x, y, sentinel float64) float64 {
	q := x / y
	if math.IsNaN(q) {
		return sentinel
	}
	return q
}

// numLines counts the number of lines that
// - are not blank
// - do not look like comments
func numLines(data []byte) int {

	data = stripComments(data)

	len := len(data)
	if len == 0 {
		return 0
	}

	count := 1
	start := 0

	pos := bytes.IndexByte(data[start:], '\n')
	for pos != -1 && start < len {
		l := data[start : start+pos+1]
		if isNotBlank(l) {
			count++
		}
		start += pos + 1
		pos = bytes.IndexByte(data[start:], '\n')
	}

	return count
}

// stripComments strips single line (//) and multi line (/* */) comments from the data
func stripComments(data []byte) []byte {

	ret := make([]byte, 0, len(data))
	s := &scanner.Scanner{}
	s.Init(bytes.NewReader(data))
	s.Error = func(_ *scanner.Scanner, _ string) {}
	s.Mode = s.Mode ^ scanner.SkipComments

	offset := 0

	tok := s.Scan()
	for tok != scanner.EOF {
		pos := s.Pos()
		if tok == scanner.Comment {
			ret = append(ret, data[offset:pos.Offset-len(s.TokenText())]...)
		} else {
			ret = append(ret, data[offset:pos.Offset]...)
		}
		offset = pos.Offset
		tok = s.Scan()
	}

	return append(ret, data[offset:]...)
}

// isNotBlank returns true if data contains at least one not-whitespace character
func isNotBlank(data []byte) bool {
	for _, r := range data {
		if !unicode.IsSpace(rune(r)) {
			return true
		}
	}
	return false
}

// adjustDefKey normalizes DefKey to be used in map.get() operations
// the following fields are used for comparison: UnitType, Unit, Path
func adjustDefKey(key graph.DefKey, unit *unit.SourceUnit) graph.DefKey {

	ret := graph.DefKey{
		Repo:     "",
		UnitType: key.UnitType,
		Unit:     key.Unit,
		Path:     key.Path,
	}

	if ret.UnitType == "" {
		ret.UnitType = unit.Type
	}
	if ret.Unit == "" {
		ret.Unit = unit.Name
	}

	return ret
}
