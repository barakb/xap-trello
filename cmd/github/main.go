package main

import (
	"net/http"
	"fmt"
	"github.com/barakb/xap-trello"
	"golang.org/x/oauth2"
	"github.com/google/go-github/github"
	githuboauth "golang.org/x/oauth2/github"

	"io/ioutil"
)

func main() {

	config := &xap_trello.Oauth2Config{}
	xap_trello.FromJSONFile(config, "github-secret.json")
	token, err := readToken()

	if err != nil{
		http.HandleFunc("/", xap_trello.HandleMain)
		http.HandleFunc("/login", xap_trello.HandleGitHubLogin)
		http.HandleFunc("/github_oauth_cb", xap_trello.HandleGitHubCallback)
		fmt.Print("Started running on http://127.0.0.1:7000\n")
		fmt.Println(http.ListenAndServe(":7000", nil))
	}else{
		oauthConf := &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			// select level of access you want https://developer.github.com/v3/oauth/#scopes
			Scopes:       []string{"user:email", "public_repo"},
			Endpoint:     githuboauth.Endpoint,
		}

		oauthClient := oauthConf.Client(oauth2.NoContext, token)
		client := github.NewClient(oauthClient)
		user, _, err := client.Users.Get("")
		if err != nil {
			fmt.Printf("client.Users.Get() faled with '%s'\n", err)
			return
		}
		fmt.Printf("Logged in as GitHub user: %s\n", *user.Login)
	}

}
func readToken() (*oauth2.Token, error) {
	jsonToken, err := ioutil.ReadFile(xap_trello.TOKEN_FILE_NAME)
	if err != nil{
		return nil, err
	}
	return xap_trello.TokenFromJSON(string(jsonToken))
}