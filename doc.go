package doc

import (
	"errors"
	"github.com/russross/blackfriday"
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
func ToHTML(format Format, source string) (html string, err error) {
	switch format {
	case Markdown:
		var out []byte
		out = blackfriday.MarkdownCommon([]byte(source))
		html = string(out)
	case ReStructuredText:
		html, err = ReStructuredTextToHTML(source)
	case Text:
	default:
		err = ErrUnhandledFormat
	}
	if err != nil || html == "" {
		html = "<pre>" + strings.TrimSpace(source) + "</pre>"
	}
	return
}

var ErrUnhandledFormat = errors.New("unhandled doc format")
