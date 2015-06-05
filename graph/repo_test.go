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
		{"foo/github.com/user/repo", ""},
		{"/foo/github.com/user/repo", ""},
		{"http://foo.com/", ""},
		{"http://foo.com/bar?baz#qux", "foo.com/bar"},
	}

	for _, test := range tests {
		got, _ := TryMakeURI(test.cloneURL)
		if test.want != got {
			t.Errorf("%q: want URI %q, got %q", test.cloneURL, test.want, got)
		}
	}
}
