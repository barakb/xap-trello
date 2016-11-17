package main

import (
	"log"
	"fmt"
	"regexp"
	"strings"
	"github.com/barakb/xap-trello"
	"flag"
	"github.com/barakb/go-trello"
)

func main() {
	listsPtr := flag.String("lists", "", "the lists in trello XAP Scrum board that should be processed, default is current sprint name from jira, use * for all and space delimiter list of strings for multiple values")
	flag.Parse()

	xapTrello, err := xap_trello.CreateXAPTrello()
	if err != nil {
		log.Fatal(err)
	}

	xapOpenJira, err := xap_trello.CreateXAPJiraOpen()
	if err != nil {
		log.Fatal(err)
	}

	board, err := xapTrello.Board("XAP Scrum")
	if err != nil {
		log.Println(err.Error())
		log.Fatal(err)
	}

	var trelloLists []trello.List
	log.Printf("listPtr is %q\n", *listsPtr)
	if *listsPtr == "*" {
		trelloLists, err = board.Lists()
	} else if *listsPtr == "" {
		log.Printf("Using active sprint %q as name for trello list\n", xapOpenJira.ActiveSprint.Name)
		trelloLists, err = board.Lists(xapOpenJira.ActiveSprint.Name)
	} else {
		names := regexp.MustCompile("\\s+,\\s+").Split(*listsPtr, -1)
		trelloLists, err = board.Lists(names...)
	}
	if err != nil {
		log.Fatal(err)
	}
	for _, aList := range trelloLists {
		log.Printf("Processing trello list %q\n", aList.Name)
		cards, err := aList.Cards()
		if err != nil {
			log.Fatal(err)
		}
		for _, card := range cards {
			if hasBugPattern(card.Name) {
				if !isAttached(card.Desc) {
						key, err := xapOpenJira.CreateBug(card.Name, card.Desc, card.Url)
						if err != nil {
							log.Printf("Failed to add jira bug for card %s, error is %s\n", card.Name, err.Error())
							continue
						}
						err = xapOpenJira.AddToActiveSprint(key)
						if err != nil {
							log.Printf("Failed to move bug:%s to sprint %s, error is %s\n", key, xapOpenJira.ActiveSprint.Name, err.Error())
							continue
						}
						newDesc := fmt.Sprintf("[:ant: %[1]s](%s/browse/%[1]s).\n\n", key, xapOpenJira.Url) + card.Desc
						card.SetDesc(newDesc)
						log.Printf("Bug:%q -> %s/browse/%s\n", card.Name, xapOpenJira.Url, key)
				}
			} else if hasFeaturePattern(card.Name) {
				if !isAttached(card.Desc) {
					key, err := xapOpenJira.CreateFeature(card.Name, card.Desc, card.Url)
					if err != nil {
						log.Printf("Failed to add jira feature for card %s, error is %s\n", card.Name, err.Error())
						continue
					}
					err = xapOpenJira.AddToActiveSprint(key)
					if err != nil {
						log.Printf("Failed to move bug:%s to sprint %s, error is %s\n", key, xapOpenJira.ActiveSprint.Name, err.Error())
						continue
					}
					newDesc := fmt.Sprintf("[:bulb: %[1]s](%s/browse/%[1]s).\n\n", key, xapOpenJira.Url) + card.Desc
					card.SetDesc(newDesc)
					log.Printf("Feature:%q -> %s/browse/%s\n", card.Name, xapOpenJira.Url, key)
				}
			} else if hasTaskPattern(card.Name) {
				//todo
			} else if key, assigned := isAttachingRequired(card.Name); assigned{
				if !isAttached(card.Desc) {
					err := xapOpenJira.AttachIssueToTrelloCard(key, card.Url)
					if err != nil {
						log.Printf("Failed to attach card %s, to issue %s, error is:%s\n", card.Name, key, err.Error())
						continue
					}
					newDesc := fmt.Sprintf("[:link: %[1]s](%s/browse/%[1]s).\n\n", key, xapOpenJira.Url) + card.Desc
					card.SetDesc(newDesc)
					log.Printf("Feature:%q (linked)-> %s/browse/%s\n", card.Name, xapOpenJira.Url, key)
					err = xapOpenJira.AddToActiveSprint(key)
					if err != nil {
						log.Printf("Failed to move card %s, to current sprint, error is:%s\n", key, err.Error())
						continue
					}

				}
			}
		}
	}

	/*
	issues, err := xapOpenJira.GetAllCurrentSprintIssues()
	if err != nil {
		log.Fatal(err)
	}
	for _, issue := range issues{
		cards, err := xapTrello.SearchMember(issue.Key)
		if err != nil{
			log.Printf("Failed to search for card with jira key %q, error is:%s", issue.Key, err.Error())
			continue
		}
		if len(cards) == 0{
			log.Printf("search for %q return %d cards %v\n", issue.Key, len(cards), cards)
		}else{
			log.Printf("search for %q return %d cards %v\n", issue.Key, len(cards), cards)
		}
	}
	*/

}

func isAttached(desc string) bool {
	//[:ant: XAP-13053](https://xap-issues.atlassian.net/browse/XAP-13053).
	re := regexp.MustCompile(`\[.*XAP\-(\d+)\]\s*\(https://xap\-issues.atlassian.net/browse/XAP\-(\d+)\)`)
	found := re.FindStringSubmatch(desc)
	if found != nil {
		return len(found) == 3 && found[1] == found[2]
	}
	return false
}

func isAttachingRequired(name string) (string, bool) {
	//XAP-13053
	re := regexp.MustCompile(`XAP\-(\d+)`)
	found := re.FindStringSubmatch(name)
	if found != nil {
		return "XAP-" + found[1], true
	}
	return "", false
}


func hasBugPattern(name string) bool {
	return strings.Contains(strings.ToLower(name), "xap-bug")
}
func hasFeaturePattern(name string) bool {
	return strings.Contains(strings.ToLower(name), "xap-feature")
}
func hasTaskPattern(name string) bool {
	return strings.Contains(strings.ToLower(name), "xap-task")
}


