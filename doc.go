package doc

import (
	"errors"
	"html"
	"strings"

	"github.com/russross/blackfriday"
)

type Format string

const (
	Text             Format = "text"
	Markdown         Format = "markdown"
	ReStructuredText Format = "rst"
)

// ToHTML converts a source document in format to an HTML string. If conversion
// fails, it returns a failsafe plaintext-to-HTML conversion and a non-nil error.
func ToHTML(format Format, source string) (htmlSource string, err error) {
	switch format {
	case Markdown:
		// Some README.md files use "~~~" instead of "```" for delimiting code
		// blocks. But "~~~" is not supported by blackfriday, so hackily replace
		// the former with the latter. See, e.g., the code blocks at
		// https://raw.githubusercontent.com/go-martini/martini/de643861770082784ad14cba4557ad68568dcc7b/README.md.
		source = strings.Replace(source, "\n~~~", "\n```", -1)

		var out []byte
		out = blackfriday.MarkdownCommon([]byte(source))
		htmlSource = string(out)
	case ReStructuredText:
		htmlSource, err = ReStructuredTextToHTML(source)
	case Text:
	default:
		err = ErrUnhandledFormat
	}
	if err != nil || htmlSource == "" {
		htmlSource = "<pre>" + strings.TrimSpace(html.EscapeString(source)) + "</pre>"
	}
	return
}

var ErrUnhandledFormat = errors.New("unhandled doc format")
