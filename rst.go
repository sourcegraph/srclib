package doc

import (
	"os"
	"os/exec"
	"strings"
)

func ReStructuredTextToHTML(txt string) (string, error) {
	cmd := exec.Command("rst2html", "--quiet")
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
