package main

import (
	"github.com/barakb/xap-trello"
	"log"
	"time"
	"gopkg.in/tylerb/graceful.v1"
	"fmt"
	"github.com/barakb/go-trello"
)

func main() {
	xap_trello.NewWatcher(func(l trello.List) bool {
		return l.Name == "Barak"
	}, 10 * time.Second)
	router := xap_trello.NewRouter()
	if err := graceful.RunWithErr(fmt.Sprintf(":%d", 6060), 10 * time.Second, router); err != nil {
		log.Fatal(err)
	}
}

