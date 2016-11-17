package xap_trello

import (
	"github.com/barakb/go-jira"
	"os"
	"fmt"
	"strings"
	"regexp"
)

type Jira struct {
	Client       *jira.Client
	ActiveSprint jira.Sprint
	IssueTypes   map[string]jira.IssueType
	Url          string
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
	if err != nil {
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
	for _, board := range boardsList.Values {
		boardsIdMap[board.Name] = fmt.Sprintf("%d", board.ID)
	}
	mainScrumBoardId := boardsIdMap["Main Scrum Board"]
	activeSprints, _, err := j.Client.Board.GetAllActiveSprints(mainScrumBoardId)
	if err != nil {
		return nil, err
	}
	if len(activeSprints) != 1 {
		return nil, fmt.Errorf("fail to find active sprint: %v\n", activeSprints)
	}
	j.ActiveSprint = activeSprints[0]
	project, _, err := j.Client.Project.Get("XAP")
	if err != nil {
		return nil, err
	}
	j.IssueTypes = map[string]jira.IssueType{}
	for _, issueType := range project.IssueTypes {
		j.IssueTypes[issueType.Name] = issueType
	}
	return j, nil
}

func (j Jira) GetAllCurrentSprintIssues() ([]jira.Issue, error) {
	issues, _, err := j.Client.Issue.Search(fmt.Sprintf("Sprint=%d", j.ActiveSprint.ID), nil)
	return issues, err
}

func (j Jira) CreateFeature(name, desc, cardUrl string) (string, error) {
	return j.createXAPIssue(name, desc, j.IssueTypes["New Feature"].ID, "Feature", cardUrl)
}

func (j Jira) CreateBug(name, desc, cardUrl string) (string, error) {
	return j.createXAPIssue(name, desc, j.IssueTypes["Bug"].ID, "BUG", cardUrl)
}

func (j Jira) createXAPIssue(name, desc, issueTypeId, issueTypeName, cardUrl string) (string, error) {
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
				Key: "XAP",
			},
			Summary: name,
			Description: summary,
		},
	}
	issue, _, err := j.Client.Issue.Create(&i)
	if err != nil {
		return "", err

	}
	err = j.AttachIssueToTrelloCard(issue.Key, cardUrl)
	return issue.Key, err
}

func (j Jira) AttachIssueToTrelloCard(key, url string) error {
	fieldId, err := j.Client.Issue.GetCustomFieldId(key, "Trello Card")
	if err != nil{
		return err
	}
	_, err = j.Client.Issue.SetCustomField(key, fieldId, url)
	return err
}

func (j Jira) AddToActiveSprint(issueKey string) error {
	_, err := j.Client.Sprint.MoveIssuesToSprint(j.ActiveSprint.ID, []string{issueKey})
	return err
}
