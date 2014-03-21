package scan

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kr/pretty"
)

func TestFiles(t *testing.T) {
	mustParseLanguages()

	tests := []struct {
		dir       string
		wantFiles map[string]*FileInfo
	}{
		{
			dir: "TestFiles",
			wantFiles: map[string]*FileInfo{
				"testdata/TestFiles/foo.go": &FileInfo{
					Language: languagesByName["Go"],
					Lib:      true,
					Analyze:  true,
					Blame:    true,
				},
			},
		},
	}

	for _, test := range tests {
		files, err := Files(filepath.Join("testdata", test.dir))
		if err != nil {
			t.Errorf("%s: Files failed: %s", test.dir, err)
			continue
		}
		clearStat(files)
		if diff := pretty.Diff(test.wantFiles, files); len(diff) > 0 {
			t.Errorf("%s: files\n%s", test.dir, strings.Join(diff, "\n"))
		}
	}
}

func clearStat(files map[string]*FileInfo) {
	for _, file := range files {
		file.Stat = nil
	}
}
