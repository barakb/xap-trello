package main

import (
	"github.com/barakb/xap-trello"
	"flag"
)

func main() {
	listsPtr := flag.Int("lists", 3, "The nuber of lists (start counting from the left) in the 'XAP Scrum' board to process")
	flag.Parse()
	xap_trello.Trello2Jira(*listsPtr, -1)

}
