package doc

import (
	"os"
	"os/exec"
	"strings"
)

var rst2html string

func init() {
	rst2html = os.Getenv("RST2HTML")
	if rst2html == "" {
		rst2html = "rst2html.py"
	}
}

func ReStructuredTextToHTML(txt string) (string, error) {
	cmd := exec.Command(rst2html, "--quiet")
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	in.Write([]byte(txt))
	in.Close()
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}

	html := string(b)
	start := strings.Index(html, "<body>") + len("<body>")
	end := strings.Index(html, "</body>")
	return strings.TrimSpace(html[start:end]), nil
}
