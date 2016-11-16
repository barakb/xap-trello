package xap_trello

import (
	"github.com/barakb/go-jira"
	"os"
	"fmt"
	"strings"
	"regexp"
)

type Jira struct {
	Client *jira.Client
	ActiveSprint jira.Sprint
	IssueTypes map[string]jira.IssueType
	Url string
}

func create(url string) (*Jira, error) {
	jiraClient, err := jira.NewClient(nil, url)
	if err != nil {
		return nil, err
	}
	res, err := jiraClient.Authentication.AcquireSessionCookie(os.Getenv("JIRA_USER"), os.Getenv("JIRA_PASSWORD"))
	if err != nil {
		return nil, err
	}
	if res == false {
		return nil, fmt.Errorf("Fail to autenticate user %s\n", os.Getenv("JIRA_USER"))
	}
	return &Jira{Client : jiraClient, Url: url}, nil
}

func CreateXAPJiraOpen() (*Jira, error) {
	j, err := create("https://xap-issues.atlassian.net")
	if err != nil{
		return nil, err
	}

	boardsListOptions := &jira.BoardListOptions{
		BoardType:      "scrum",
		ProjectKeyOrID: "XAP",
	}
	boardsList, _, err := j.Client.Board.GetAllBoards(boardsListOptions)
	if err != nil {
		return nil, err
	}
	boardsIdMap := map[string]string{}
	for _, board := range boardsList.Values{
		boardsIdMap[board.Name] = fmt.Sprintf("%d", board.ID)
	}
	mainScrumBoardId := boardsIdMap["Main Scrum Board"]
	activeSprints, _, err := j.Client.Board.GetAllActiveSprints(mainScrumBoardId)
	if err != nil {
		return nil, err
	}
	if len(activeSprints) != 1{
		return nil, fmt.Errorf("fail to find active sprint: %v\n", activeSprints)
	}
	j.ActiveSprint = activeSprints[0]

	project, _, err := j.Client.Project.Get("XAP")
	if err != nil {
		return nil, err
	}
	j.IssueTypes = map[string]jira.IssueType{}
	for _, issueType := range project.IssueTypes{
		j.IssueTypes[issueType.Name] = issueType
	}
	return j, nil
}




func (j Jira) CreateFeature(name, desc string) (string, error) {
	return j.createXAPIssue(name, desc, j.IssueTypes["New Feature"].ID, "Feature")
}

func (j Jira) CreateBug(name, desc string) (string, error){
	return j.createXAPIssue(name, desc, j.IssueTypes["Bug"].ID, "BUG")
}

func (j Jira) createXAPIssue(name, desc, issueTypeId, issueTypeName string) (string, error){
	var summary = desc
	if summary == "" {
		summary = name
	}
	summary = strings.TrimLeft(regexp.MustCompile(fmt.Sprintf("(?i)XAP-%s", issueTypeName)).ReplaceAllLiteralString(summary, ""), " ")
	i := jira.Issue{
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				ID: issueTypeId,
			},
			Project: jira.Project{
				ID: "10060",  //todo remove this hardcoded
			},
			Summary: name,
			Description: summary,
		},
	}
	issue, _, err := j.Client.Issue.Create(&i)
	if err != nil {
		return "", err

	}
	return issue.Key, nil
}

func (j Jira) AddToActiveSprint(issueKey string) error {
	_, err := j.Client.Sprint.MoveIssuesToSprint(j.ActiveSprint.ID, []string{issueKey})
	return err
}
