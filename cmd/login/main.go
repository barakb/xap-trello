package main

import (
	"github.com/gorilla/sessions"
	"os"
	"github.com/gorilla/mux"
	"net/http"
	"time"
	"log"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"

	"fmt"
	"github.com/google/go-github/github"
)

var (
	store = sessions.NewCookieStore([]byte("fobar5something-very-secret"))
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		// select level of access you want https://developer.github.com/v3/oauth/#scopes
		Scopes:       []string{"user:email"},
		Endpoint:     githuboauth.Endpoint,
	}
	// random string for oauth2 API calls to protect against CSRF
	oauthStateString = "foobarrandomfoobar9"
)

const htmlIndex = `<html><body>
Logged in with <a href="/logout">GitHub</a>
</body></html>
`

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", HandleMain)
	r.HandleFunc("/github_oauth_cb", HandleGitHubCallback)
	r.HandleFunc("/secured", HandleSecured)
	r.HandleFunc("/logout", HandleLogout)
	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:7000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())

}

func HandleMain(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user, ok := session.Values["user"]
	if ok && user != ""{
		log.Printf("Found user %s\n", user)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlIndex))
	} else {
		log.Println("Didn't Found username")
		url := oauthConf.AuthCodeURL(oauthStateString, oauth2.AccessTypeOnline)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func HandleSecured(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleSecured")
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Save(r, w)
	user, ok := session.Values["user"]
	if ok {
		log.Printf("Found user %s\n", user)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlIndex))
	} else {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleLogout")
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["user"] = ""
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

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

	oauthClient := oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	user, _, err := client.Users.Get("")
	if err != nil {
		fmt.Printf("client.Users.Get() faled with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	session, err := store.Get(r, "session-name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["user"] = *user.Login
	fmt.Printf("Logged in as GitHub user: %s\n", *user.Login)
	session.Save(r, w)

	grants, _, err := client.Authorizations.ListGrants()
	if err != nil {
		log.Printf("Got error while getting grants: %s\n", err.Error())
	}
	log.Printf("Grants %+v\n", grants)
	for _, grant := range grants{
		log.Printf("Grant %+v\n", grant)
	}

	http.Redirect(w, r, "/secured", http.StatusTemporaryRedirect)
}

