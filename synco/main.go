package main

import (
	"log"
	"os"
	"fmt"
	"strings"
	"regexp"
	"github.com/barakb/go-trello"
	"github.com/barakb/go-jira"
	"github.com/davecgh/go-spew/spew"
)

var featureRegex = regexp.MustCompile(`(?i)XAP-FEATURE`)
var bugRegex = regexp.MustCompile(`(?i)XAP-BUG`)
var taskRegex = regexp.MustCompile(`(?i)XAP-TASK`)

func main() {
	trelloAppToken := os.Getenv("trelloAppToken")
	trelloClient, err := trello.NewAuthClient(os.Getenv("trelloAppKey"), &trelloAppToken)
	if err != nil {
		log.Fatal(err)
	}
	u, err := trelloClient.Member("barakbarorion")
	if err != nil {
		log.Fatal(err)
	}

	jiraClient := createJiraClient()

	issue, _, err := jiraClient.Issue.Get("XAP-12352")
	spew.Dump(issue)
	storyPointsFieldId, err := jiraClient.Issue.GetCustomFieldId("XAP-12352", "Story Points")
	if err != nil {
		log.Fatalf("Failed to find story points field id: %s", err)
	}
	log.Printf("story points field id:%s\n", storyPointsFieldId)
	storyPoints := int(issue.Fields.Unknowns[storyPointsFieldId].(float64))
	log.Printf("story points is:%d\n", storyPoints)

	jiraClient.Issue.GetCustomFieldId("XAP-12352", storyPointsFieldId)


	metaInfo, _, _ := jiraClient.Issue.GetCreateMeta("XAP")
	if err != nil {
		log.Fatalf("Expected nil error but got %s", err)
	}
	log.Printf("metaInfo is:%+v\n", *metaInfo)
	//spew.Dump(*metaInfo)
	project := metaInfo.GetProjectWithKey("XAP")
	for _, issueType := range project.IssueTypes{
		fields,_ := issueType.GetAllFields()
		log.Printf("Issue:%s, fields %v\n",issueType.Name, fields)
	}
	boardsListOptions := &jira.BoardListOptions{
		BoardType:      "scrum",
		ProjectKeyOrID: "XAP",
	}
	boardsList, _, err := jiraClient.Board.GetAllBoards(boardsListOptions)
	if err != nil {
		log.Fatal(err)
	}
	boardsIdMap := map[string]string{}
	for _, board := range boardsList.Values{
		log.Printf("Board name:%s, id:%d\n", board.Name, board.ID)
		boardsIdMap[board.Name] = fmt.Sprintf("%d", board.ID)
	}
	mainScrumBoardId := boardsIdMap["Main Scrum Board"]
	activeSprints, _, err := jiraClient.Board.GetAllActiveSprints(mainScrumBoardId)
	if err != nil {
		log.Fatal(err)
	}
	if len(activeSprints) != 1{
		log.Fatalf("fail to find active sprint: %v\n", activeSprints)
	}
	activeSprint := activeSprints[0]

	board, err := getBoard("XAP Scrum", u)
	if err != nil {
		log.Fatal(err)
	}

	lists, err := board.Lists()
	if err != nil {
		log.Fatal(err)
	}

	for _, aList := range lists {
		if aList.Name == "Barak" {
			cards, err := aList.Cards()
			if err != nil {
				log.Fatal(err)
			}
			for _, card := range cards {
				if hasBugPattern(card.Name) {
					log.Printf("card.Name: %s is bug\n", card.Name)
					if !isBugResolved(card.Desc) {
						key, err := addJiraBug(jiraClient, card)
						if err != nil {
							log.Printf("Failed to add jira issue for card %s, error is %s\n", card.Name, err.Error())
							continue
						}
						err = addJiraIssueToSprint(jiraClient, key, activeSprint)
						if err != nil{
							log.Printf("Failed to move issue:%s to sprint %s, error is %s\n", key, activeSprint.Name, err.Error())
						}
						newDesc := fmt.Sprintf("[:ant: %[1]s](https://xap-issues.atlassian.net/browse/%[1]s).\n\n", key) + card.Desc
						card.SetDesc(newDesc)

					}
				} else if hasFeaturePattern(card.Name) {
					log.Printf("card.Name: %s is feature\n", card.Name)
				} else if hasTaskPattern(card.Name) {
					log.Printf("card.Name: %s is feature\n", card.Name)
				}

			}
		}
	}
}
func addJiraIssueToSprint(client *jira.Client, issueKey string, sprint jira.Sprint) error {
	_, err := client.Sprint.MoveIssuesToSprint(sprint.ID, []string{issueKey})
	return err
}

func addJiraBug(jiraClient *jira.Client, card trello.Card) (string, error) {

	var summary = card.Desc
	if summary == "" {
		summary = card.Name
	}
	summary = strings.TrimLeft(regexp.MustCompile(`(?i)XAP-BUG`).ReplaceAllLiteralString(summary, ""), " ")
	i := jira.Issue{
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				ID: "1",
			},
			Project: jira.Project{
				ID: "10060",
			},
			Summary: card.Name,
			Description: summary,
		},
	}
	issue, _, err := jiraClient.Issue.Create(&i)
	if err != nil {
		return "", err

	}
	return issue.Key, nil
}

func isBugResolved(desc string) bool {
	//[BUG XAP-13053](https://xap-issues.atlassian.net/browse/XAP-13053).
	re := regexp.MustCompile(`\[:ant:\s+XAP\-(\d+)\]\s*\(https://xap\-issues.atlassian.net/browse/XAP\-(\d+)\)`)
	found := re.FindStringSubmatch(desc)
	if found != nil {
		return len(found) == 3 && found[1] == found[2]
	}
	return false

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

func getBoard(name string, member *trello.Member) (b trello.Board, e error) {
	boards, err := member.Boards()
	if err != nil {
		return b, err
	}
	for _, board := range boards {
		if board.Name == name {
			return board, nil
		}
	}
	return b, fmt.Errorf("failed to find board %q\n", name)
}

func createJiraClient() *jira.Client {
	jiraClient, err := jira.NewClient(nil, "https://xap-issues.atlassian.net")
	if err != nil {
		panic(err)
	}
	res, err := jiraClient.Authentication.AcquireSessionCookie(os.Getenv("JIRA_USER"), os.Getenv("JIRA_PASSWORD"))
	if err != nil || res == false {
		fmt.Printf("Result: %v\n", res)
		panic(err)
	}
	return jiraClient
}

