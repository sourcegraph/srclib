package ruby

import (
	"testing"
)

func TestResolveGem(t *testing.T) {
	tests := []struct {
		gemName  string
		cloneURL string
		err      error
	}{
		{"rails", "git://github.com/rails/rails", nil},
		{"sample_ruby_gem", "https://github.com/sgtest/sample_ruby_gem", nil},
		// twice so we test the cache
		{"sample_ruby_gem", "https://github.com/sgtest/sample_ruby_gem", nil},
		{"gemdoesntexist923432", "", ErrNoGemFound},
		{"gemdoesntexist923432", "", ErrNoGemFound},
		{"gemdoesnt#$ ", "", ErrGemInvalidName},
		{"", "", ErrGemEmptyName},
	}
	for _, test := range tests {
		cloneURL, err := ResolveGem(test.gemName)
		if test.err != err {
			t.Errorf("%s: want err %v, got %v", test.gemName, test.err, err)
			continue
		}
		if test.cloneURL != cloneURL {
			t.Errorf("%s: want cloneURL %s, got %s", test.gemName, test.cloneURL, cloneURL)
		}
	}
}
