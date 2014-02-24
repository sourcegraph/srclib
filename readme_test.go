package doc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sourcegraph/vcsserver"
	"github.com/sqs/gorp"
	"sourcegraph.com/sourcegraph/config"
	"sourcegraph.com/sourcegraph/db"
	"sourcegraph.com/sourcegraph/repo"
)

type getFormattedReadmeTest struct {
	repo                 *repo.Repository
	remoteReadmeFilename string
	remoteReadmeContents string
	wantReadme           string
	wantErr              error
}

func TestGetFormattedReadme(t *testing.T) {
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

	db.Connect()
	repo.MapDB(&db.DB)
	for label, test := range tests {
		func() {
			tx, _ := db.DB.DbMap.Begin()
			defer tx.Rollback()
			tx.Insert(test.repo)
			testGetFormattedReadme(t, tx, label, test)
		}()
	}
}

func testGetFormattedReadme(t *testing.T, tx gorp.SqlExecutor, label string, test getFormattedReadmeTest) {
	u, err := url.Parse(test.repo.CloneURL)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	handled := false
	handlerPath := vcsserver.BatchFilesURI(string(test.repo.VCS), u, test.repo.RevSpecOrDefault(), []string{}).Path
	mux.HandleFunc(handlerPath, func(w http.ResponseWriter, _ *http.Request) {
		handled = true
		w.Header().Set("x-batch-file", test.remoteReadmeFilename)
		io.WriteString(w, test.remoteReadmeContents)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	origVCSMirrorURL := config.VCSMirrorURL
	config.VCSMirrorURL, _ = url.Parse(server.URL)
	defer func() {
		config.VCSMirrorURL = origVCSMirrorURL
	}()

	readme, err := GetFormattedReadme(tx, test.repo)
	if err != test.wantErr {
		t.Errorf("%s: GetFormattedReadme: want err == %v, got %v", label, test.wantErr, err)
		return
	}
	if test.wantReadme != readme {
		t.Errorf("%s: want readme == %q, got %q", label, test.wantReadme, readme)
	}

	if !handled {
		t.Errorf("%s: readme handler never called")
	}
}
