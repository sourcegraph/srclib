package srcgraph

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"sourcegraph.com/sourcegraph/config2"
	"sourcegraph.com/sourcegraph/repo"
)

func push(args []string) {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	r := AddRepositoryFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: `+Name+` push [options]

Updates a repository and related information on Sourcegraph. Graph data for this
repository and commit that was previously uploaded using the "`+Name+`" tool
will be used; if none exists, Sourcegraph will build the repository remotely.

The options are:
`)
		fs.PrintDefaults()
		os.Exit(1)
	}
	fs.Parse(args)

	url := config2.BaseAPIURL.ResolveReference(&url.URL{
		Path: fmt.Sprintf("repositories/%s/commits/%s/build", repo.MakeURI(r.CloneURL), r.CommitID),
	})
	req, err := http.NewRequest("PUT", url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatalf("Push failed: HTTP %s (%s).", resp.Status, string(body))
	}

	log.Printf("Push succeeded. The repository will be updated on Sourcegraph soon.")
}
