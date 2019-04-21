package xap_trello

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
	"io/ioutil"
	"net/http"
)

var (
	// You must register the app at https://github.com/settings/applications
	// Set callback to http://127.0.0.1:7000/github_oauth_cb
	// Set ClientId and ClientSecret to
	oauthConf = &oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		// select level of access you want https://developer.github.com/v3/oauth/#scopes
		Scopes:   []string{"user:email", "public_repo"},
		Endpoint: githuboauth.Endpoint,
	}
	// random string for oauth2 API calls to protect against CSRF
	oauthStateString = "thisshouldberandom"
)

const htmlIndex = `<html><body>
Logged in with <a href="/login">GitHub</a>
</body></html>
`
const TOKEN_FILE_NAME = "github-token.json"

// /
func HandleMain(w http.ResponseWriter, r *http.Request) {
	config := &Oauth2Config{}
	if err := FromJSONFile(config, "github-secret.json"); err != nil {
		fmt.Printf("Error in reading github client secret from file github-secret.json error is: %s\n", err.Error())
	}

	oauthConf.ClientID, oauthConf.ClientSecret = config.ClientID, config.ClientSecret

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlIndex))
}

// /login
func HandleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	url := oauthConf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// /github_oauth_cb. Called by github after authorization is granted
func HandleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		fmt.Printf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		fmt.Printf("oauthConf.Exchange() failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	jsonToken, err := tokenToJSON(token)
	if err != nil {
		fmt.Printf("convert token to json failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	err = ioutil.WriteFile(TOKEN_FILE_NAME, []byte(jsonToken), 0666)
	if err != nil {
		fmt.Printf("failed to write token to file %s, error is:'%s'\n", TOKEN_FILE_NAME, err)
	}
	oauthClient := oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		fmt.Printf("client.Users.Get() faled with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	fmt.Printf("Logged in as GitHub user: %s\n", *user.Login)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func tokenToJSON(token *oauth2.Token) (string, error) {
	if d, err := json.Marshal(token); err != nil {
		return "", err
	} else {
		return string(d), nil
	}
}

func TokenFromJSON(jsonStr string) (*oauth2.Token, error) {
	var token oauth2.Token
	if err := json.Unmarshal([]byte(jsonStr), &token); err != nil {
		return nil, err
	}
	return &token, nil
}

type Oauth2Config struct {
	ClientID     string
	ClientSecret string
}

func ToJSONFile(val interface{}, filename string) error {
	if bytes, err := json.Marshal(val); err != nil {
		return err
	} else {
		return ioutil.WriteFile(filename, []byte(bytes), 0666)
	}
}

func FromJSONFile(val interface{}, filename string) error {
	jsonBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, val)
}

func ReadGithubToken() (*oauth2.Token, error) {
	jsonToken, err := ioutil.ReadFile(TOKEN_FILE_NAME)
	if err != nil {
		return nil, err
	}
	return TokenFromJSON(string(jsonToken))
}
