package doc

import (
	"errors"
	"github.com/russross/blackfriday"
	"html"
	"strings"
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
