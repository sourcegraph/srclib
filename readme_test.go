package doc

import (
	"testing"

	"github.com/sourcegraph/go-vcs/vcs"
	vcs_testing "github.com/sourcegraph/go-vcs/vcs/testing"
	"github.com/sourcegraph/vcsstore/vcsclient"
	"sourcegraph.com/sourcegraph/srcgraph/repo"
)

type getFormattedReadmeTest struct {
	repo           *repo.Repository
	readmeFilename string
	readmeContents string

	wantFormattedReadme string
	wantErr             error
}

func TestReadmeFormatter_GetFormattedReadme(t *testing.T) {
	fooRepo := &repo.Repository{
		URI:      "github.com/foo/bar",
		CloneURL: "git://github.com/foo/bar.git",
		VCS:      "git",
	}

	tests := map[string]getFormattedReadmeTest{
		"markdown": {
			fooRepo,
			"README.md",
			"hello\n=====\n\nworld",
			"<h1>hello</h1>\n\n<p>world</p>\n",
			nil,
		},
		"rst": {
			fooRepo,
			"README.rst",
			"======\nhello\n======\n\nworld",
			"<div class=\"document\" id=\"hello\">\n<h1 class=\"title\">hello</h1>\n\n<p>world</p>\n</div>",
			nil,
		},
		"text": {
			fooRepo,
			"README",
			"hello world",
			"<pre>hello world</pre>",
			nil,
		},
	}

	for label, test := range tests {
		testGetFormattedReadme(t, label, test)
	}
}

func testGetFormattedReadme(t *testing.T, label string, test getFormattedReadmeTest) {
	o := vcsclient.MockRepositoryOpener{
		Return: vcs_testing.MockRepository{
			ResolveBranch_: func(name string) (vcs.CommitID, error) { return "abcd", nil },
			FileSystem_: func(at vcs.CommitID) (vcs.FileSystem, error) {
				return vcs_testing.MapFS(map[string]string{test.readmeFilename: test.readmeContents}), nil
			},
		},
	}
	rf := NewReadmeFormatter(o)

	readme, err := rf.GetFormattedReadme(test.repo)
	if err != test.wantErr {
		t.Errorf("%s: GetFormattedReadme: want err == %v, got %v", label, test.wantErr, err)
		return
	}
	if test.wantFormattedReadme != readme {
		t.Errorf("%s: got formatted readme == %q, want %q", label, readme, test.wantFormattedReadme)
	}
}
