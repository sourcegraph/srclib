package src

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/net/context"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/srclib"
	"sourcegraph.com/sqs/pbtypes"
)

var userAuthFile = filepath.Join(os.Getenv("HOME"), ".src-auth")

// userAuth holds user auth credentials keyed on API endpoint
// URL. It's typically saved in a file named by userAuthFile.
type userAuth map[string]*userEndpointAuth

// userEndpointAuth holds a user's authentication credentials for a
// srclib endpoint.
type userEndpointAuth struct {
	Key string // API key
}

// readUserAuth attempts to read a userAuth struct from the
// userAuthFile. It is not considered an error if the userAuthFile
// doesn't exist; in that case, an empty userAuth and a nil error is
// returned.
func readUserAuth() (userAuth, error) {
	f, err := os.Open(userAuthFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ua userAuth
	if err := json.NewDecoder(f).Decode(&ua); err != nil {
		return nil, err
	}
	return ua, nil
}

// writeUserAuth writes ua to the userAuthFile.
func writeUserAuth(a userAuth) error {
	f, err := os.Create(userAuthFile)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(a)
}

func init() {
	_, err := CLI.AddCommand("login",
		"log in to Sourcegraph.com",
		"The login command logs into Sourcegraph.com using your UID and API key. To obtain these values, sign up or log into Sourcegraph.com, then go to the 'Integrations' page in your user settings.",
		&loginCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = CLI.AddCommand("whoami",
		"show logged-in user login and info",
		"The whoami command prints the username and other information about the user authenticated by a previous call to `src login`.",
		&whoamiCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type LoginCmd struct {
	Key string `long:"key" description:"Sourcegraph API key" required:"yes" env:"SRC_KEY"`

	NoVerify bool `long:"no-verify" description:"don't verify login credentials by attempting to log in remotely"`
}

var loginCmd LoginCmd

func (c *LoginCmd) Execute(args []string) error {
	a, err := readUserAuth()
	if err != nil {
		return err
	}
	if a == nil {
		a = userAuth{}
	}

	ua := userEndpointAuth{Key: c.Key}
	endpointURL := getEndpointURL()

	if !c.NoVerify {
		// Overwrite Credentials to only have the API key, so we know
		// we're testing only the API key's validity.
		Credentials = CredentialOpts{APIKey: c.Key}
		authedAPIClient := Client()
		authInfo, err := authedAPIClient.UserAuth.Identify(context.TODO(), &pbtypes.Void{})
		if err != nil {
			log.Fatalf("Error verifying auth credentials with endpoint %s: %s.", endpointURL, err)
		}
		user, err := authedAPIClient.Users.Get(context.TODO(), &sourcegraph.UserSpec{UID: authInfo.UID})
		if err != nil {
			log.Fatalf("Error getting user with UID %d: %s.", authInfo.UID, err)
		}
		log.Printf("# Logged into %s as UID %d (%s) using API key.", endpointURL, user.UID, user.Login)
	}

	a[endpointURL.String()] = &ua
	if err := writeUserAuth(a); err != nil {
		return err
	}
	log.Printf("# Credentials saved to %s.", userAuthFile)
	return nil
}

type WhoamiCmd struct {
}

var whoamiCmd WhoamiCmd

func (c *WhoamiCmd) Execute(args []string) error {
	a, err := readUserAuth()
	if err != nil {
		return err
	}
	endpointURL := getEndpointURL()
	ua := a[endpointURL.String()]
	if ua == nil {
		log.Fatalf("# No authentication info set for %s (use `%s login` to authenticate)", endpointURL, srclib.CommandName)
	}

	cl := Client()

	authInfo, err := cl.UserAuth.Identify(context.TODO(), &pbtypes.Void{})
	if err != nil {
		log.Fatalf("Error verifying auth credentials with endpoint %s: %s.", endpointURL, err)
	}
	user, err := cl.Users.Get(context.TODO(), &sourcegraph.UserSpec{UID: authInfo.UID})
	if err != nil {
		log.Fatalf("Error getting user with UID %d: %s.", authInfo.UID, err)
	}
	log.Printf("%s (UID %d) on %s (verified remotely)", user.Login, user.UID, endpointURL)

	return nil
}
