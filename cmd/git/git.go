package main

import (
	"github.com/barakb/xap-trello"
	"log"
)

func main() {
	token, err := xap_trello.ReadGithubToken()

	git := xap_trello.NewGitRepository("/home/barakbo/tmp/t/imc-sprints", "https://github.com/barakb/imc-sprints.git", token.AccessToken)
	git.Init()
	//err :=  git.Log()
	//if err != nil{
	//	log.Println(err)
	//}
	//time.Sleep(100 * time.Millisecond)
	//err := git.Add("-u")
	//if err != nil{
	//	log.Println(err)
	//}
	//err = git.Commit(fmt.Sprintf("Automatic update at %v", time.Now()))
	//if err != nil{
	//	log.Println(err)
	//}
	//
	err = git.Rebase()
	if err != nil{
		log.Println(err)
	}

	err = git.Push()
	if err != nil{
		log.Println(err)
	}


}



