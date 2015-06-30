package graph

import "testing"

func TestTryMakeURI(t *testing.T) {
	tests := []struct {
		cloneURL string
		want     string
	}{
		{"https://github.com/user/repo", "github.com/user/repo"},
		{"git://github.com/user/repo", "github.com/user/repo"},
		{"http://bitbucket.org/user/repo", "bitbucket.org/user/repo"},
		{"https://bitbucket.org/user/repo", "bitbucket.org/user/repo"},
		{"bitbucket.org/user/repo", "bitbucket.org/user/repo"},
		{"", ""},
		{"http://sourcegraph.com/user/repo", "sourcegraph.com/user/repo"},
		{"ssh://hg@bitbucket.org/org/repo", "bitbucket.org/org/repo"},
		{"https://user@bitbucket.org/org/repo", "bitbucket.org/org/repo"},
		{"/foo/bar", ""},
		{"https://gitrepos/foo/bar", "gitrepos/foo/bar"},
		{"gitrepos/foo/bar", "gitrepos/foo/bar"},
		{"gitrepos/", ""},
		{"/gitrepos/foo/bar", ""},
		{"http://foo.com/", ""},
		{"http://foo.com/bar?baz#qux", "foo.com/bar"},
		{"git@foo.com:bar", "foo.com/bar"},
		{"git@foo.com:bar/baz.qux", "foo.com/bar/baz.qux"},
		{"git@foo:bar.git", "foo/bar"},
	}

	for _, test := range tests {
		got, err := TryMakeURI(test.cloneURL)
		if test.want != "" && err != nil {
			t.Errorf("%s: error: %s", test.cloneURL, err)
			continue
		}
		if test.want == "" {
			continue
		}
		if test.want != got {
			t.Errorf("%q: want URI %q, got %q", test.cloneURL, test.want, got)
		}
	}
}
