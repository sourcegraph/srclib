package python

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"sourcegraph.com/sourcegraph/doc"
)

var rstDirectiveRegex = regexp.MustCompile(`\.\. [a-zA-Z]+:: `)

// Properly format python documentation by doing the following:
// Check if txt is rST. If so, compile into html, otherwise return html approximation to plaintext.
func formatDocs(txt string) string {
	txt = doc.EscapeUnprintable(txt)

	var format doc.Formatter
	if strings.Contains(txt, "~~~") || len(rstDirectiveRegex.FindStringIndex(txt)) > 0 {
		format = doc.ReStructuredText

		var err error
		txt, err = rstPreprocess(txt)
		if err != nil {
			log.Printf("rstPreprocess failed: %s", err)
		}
	} else if strings.Contains(txt, "====") {
		format = doc.Markdown
	} else {
		format = doc.Text
	}

	html, err := doc.ToHTML(format, []byte(txt))
	if err != nil {
		log.Printf("doc.ToHTML failed: %s", err)
	}
	return string(html)
}

var indentMatcher = regexp.MustCompile("(\\s*)[^\\s].*")
var rstRoleMatcher = regexp.MustCompile(":[A-Za-z]+:`([^`]+)`")

// Preprocess docstring before it can be fed to rst2html.  This is
// necessary, because sometimes indentations in docstrings are
// formatted to look good with code, rather than be valid rst.
func rstPreprocess(txt string) (string, error) {
	lines := strings.Split(txt, "\n")

	globalIndent := ""
	foundFirstIndent := false
	for l, line := range lines {
		if l == 0 && !strings.HasPrefix(line, "..") {
			continue
		} else if strings.TrimSpace(line) == "" {
			continue
		}

		matches := indentMatcher.FindStringSubmatch(line)
		if matches == nil || len(matches) < 2 {
			return "", fmt.Errorf("Could not process line: %s", line)
		}

		if !foundFirstIndent {
			foundFirstIndent = true
			globalIndent = matches[1]
		} else {
			if len(globalIndent) > len(matches[1]) {
				globalIndent = matches[1]
			}
		}
	}

	for l, line := range lines {
		if strings.HasPrefix(line, globalIndent) {
			lines[l] = line[len(globalIndent):]
		}
	}

	indentNormalizedTxt := strings.Join(lines, "\n")
	roleRemovedTxt := rstRoleMatcher.ReplaceAllString(indentNormalizedTxt, ":code:`$1`")

	return roleRemovedTxt, nil
}
