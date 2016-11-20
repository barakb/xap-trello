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
	listsPtr := flag.Int("lists", 3, "The nuber of lists (start counting from the left) in the 'XAP Scrum' board to process")
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

	trelloLists, err := board.Lists()
	if err != nil {
		log.Fatal(err)
	}
	var trelloCardByJiraKey = map[string]trello.Card{}
	for n, aList := range trelloLists {
		if *listsPtr <= n{
			break
		}
		log.Printf("Processing trello list %q\n", aList.Name)
		cards, err := aList.Cards()
		if err != nil {
			log.Fatal(err)
		}
		for _, card := range cards {
			if hasBugPattern(card.Name) {
				if key, ok := isAttached(card.Desc); !ok {
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
						trelloCardByJiraKey[key] = card
						log.Printf("Bug:%q -> %s/browse/%s\n", card.Name, xapOpenJira.Url, key)
				}else{
					trelloCardByJiraKey[key] = card
					err = xapOpenJira.AddToActiveSprint(key)
					if err != nil {
						log.Printf("Failed to move card %s, to current sprint, error is:%s\n", key, err.Error())
					}
				}
			} else if hasFeaturePattern(card.Name) {
				if key, ok := isAttached(card.Desc); !ok {
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
					trelloCardByJiraKey[key] = card
					log.Printf("Feature:%q -> %s/browse/%s\n", card.Name, xapOpenJira.Url, key)
				}else{
					trelloCardByJiraKey[key] = card
					err = xapOpenJira.AddToActiveSprint(key)
					if err != nil {
						log.Printf("Failed to move card %s, to current sprint, error is:%s\n", key, err.Error())
					}
				}
			} else if hasTaskPattern(card.Name) {
				//todo
			} else if key, assigned := isAttachingRequired(card.Name); assigned{
				trelloCardByJiraKey[key] = card
				if _, ok := isAttached(card.Desc); !ok {
					err := xapOpenJira.AttachIssueToTrelloCard(key, card.Url)
					if err != nil {
						log.Printf("Failed to attach card %s, to issue %s, error is:%s\n", card.Name, key, err.Error())
						continue
					}
					newDesc := fmt.Sprintf("[:link: %[1]s](%s/browse/%[1]s).\n\n", key, xapOpenJira.Url) + card.Desc
					card.SetDesc(newDesc)
					log.Printf("Feature:%q (linked)-> %s/browse/%s\n", card.Name, xapOpenJira.Url, key)
				}
				err = xapOpenJira.AddToActiveSprint(key)
				if err != nil {
					log.Printf("Failed to move card %s, to current sprint, error is:%s\n", key, err.Error())
				}

			}
		}
	}


	issues, err := xapOpenJira.GetAllCurrentSprintIssues()
	if err != nil {
		log.Fatal(err)
	}
	for _, issue := range issues{
		if _, ok := trelloCardByJiraKey[issue.Key]; !ok{
			fmt.Printf("Jira sprint issue %s is not in first 3 trello lists, moving to backlog\n", issue.Key)
			_, err := xapOpenJira.Client.Sprint.MoveIssuesToBackLog(issue.Key)
			if err != nil{
				log.Printf("Failed to move issue %s to backlog, error is: %s\n", issue.Key, err.Error())
			}
		}
		/*
                transitions, _, err := xapOpenJira.Client.Issue.GetTransitions(issue.ID)
		if err != nil{
			log.Printf("Failed to get transition for issue %s, error is %s\n", issue.Key, err.Error())
			continue
		}
		for _, transition := range transitions{
			log.Printf("Issue %s has possible transition %+v\n", issue.Key, transition)
		}
		cards, err := xapTrello.SearchMember(issue.Key)
		if err != nil{
			//log.Printf("Failed to search for card with jira key %q, error is:%s", issue.Key, err.Error())
			continue
		}
		if len(cards) == 0{
			//log.Printf("search for %q return %d cards %v\n", issue.Key, len(cards), cards)
		}else{
			//log.Printf("search for %q return %d cards %v\n", issue.Key, len(cards), cards)
		}
		*/
	}


}

func isAttached(desc string) (string, bool) {
	//[:ant: XAP-13053](https://xap-issues.atlassian.net/browse/XAP-13053).
	re := regexp.MustCompile(`\[.*XAP\-(\d+)\]\s*\(https://xap\-issues.atlassian.net/browse/XAP\-(\d+)\)`)
	found := re.FindStringSubmatch(desc)
	if found != nil {
		return found[1], len(found) == 3 && found[1] == found[2]
	}
	return "", false
}

func isAttachingRequired(name string) (string, bool) {
	//XAP-13053
	re := regexp.MustCompile(`XAP\-(\d+)`)
	found := re.FindStringSubmatch(name)
	if found != nil {
		return found[0], true
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


