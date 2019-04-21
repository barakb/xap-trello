package main

import (
	"fmt"
	xap_trello "github.com/barakb/xap-trello"
	"gopkg.in/tylerb/graceful.v1"
	"log"
	"time"
)

func main() {
	//xapOpenJira, err := xap_trello.CreateXAPJiraOpen()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//xapTrello, err := xap_trello.CreateXAPTrello()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//var burndown = xap_trello.Burndown{Trello:xapTrello, Sprint:xapOpenJira.ActiveSprint}
	//log.Println("calling ScanLoop")
	//burndown.ScanLoop(2 * time.Second)
	//log.Println("Done")

	xap_trello.InitRouters()
	router := xap_trello.NewRouter()
	if err := graceful.RunWithErr(fmt.Sprintf(":%d", 6060), 10*time.Second, router); err != nil {
		log.Fatal(err)
	}
}
