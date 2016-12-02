package xap_trello

import (
	"github.com/barakb/go-jira"
	"os"
	"fmt"
	"strings"
	"regexp"
	"log"
)

type Jira struct {
	Client       *jira.Client
	ActiveSprint jira.Sprint
	IssueTypes   map[string]jira.IssueType
	Url          string
	MainScrumBoardId int
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
		if board.Name == "Main Scrum Board" {
			j.MainScrumBoardId = board.ID
		}
	}

	//start := time.Date(2016, 11, 27, 0, 0, 0, 0, time.UTC)
	//end := time.Date(2016, 12, 1, 0, 0, 0, 0, time.UTC)
	//msb, _ := strconv.Atoi(mainScrumBoardId)
	//fmt.Printf("creating sprint from %s to %s at board %d\n", start, end, msb)
	//s, _, e := j.Client.Board.CreateSprint("12.1-M7", start, end,  msb)
	//
	//if e != nil{
	//	return nil, err
	//}
	//
	//fmt.Printf("Sprint is %+v\n", s)

	activeSprints, _, err := j.Client.Board.GetAllActiveSprints(fmt.Sprintf("%d", j.MainScrumBoardId))
	if err != nil {
		return nil, err
	}
	if len(activeSprints) != 1 {
		fmt.Printf("fail to find active sprint: %v\n", activeSprints)
		//return nil, fmt.Errorf("fail to find active sprint: %v\n", activeSprints)
	}else {
		j.ActiveSprint = activeSprints[0]
	}
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

func (j Jira) GetAllSprintIssues(sprintId int) ([]jira.Issue, error) {
	issues, _, err := j.Client.Issue.Search(fmt.Sprintf("Sprint=%d", sprintId), nil)
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
	summary = regexp.MustCompile("(?i)XAP-[^ ]+").ReplaceAllLiteralString(summary, "")
	summary = strings.TrimLeft(regexp.MustCompile("\\([0-9.]+\\)").ReplaceAllLiteralString(summary, ""), " ")
	summary = strings.TrimLeft(regexp.MustCompile("\\{[0-9.]+\\}").ReplaceAllLiteralString(summary, ""), " ")
	summary = strings.TrimSpace(summary)
	name = regexp.MustCompile("(?i)XAP-[^ ]+").ReplaceAllLiteralString(name, "")
	name = strings.TrimLeft(regexp.MustCompile("\\([0-9.]+\\)").ReplaceAllLiteralString(name, ""), " ")
	name = strings.TrimLeft(regexp.MustCompile("\\{[0-9.]+\\}").ReplaceAllLiteralString(name, ""), " ")
	name = strings.TrimSpace(name)
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
	return j.AddToSprint(issueKey, j.ActiveSprint.ID)
}

func (j Jira) AddToSprint(issueKey string, activeSprintId int) error {
	log.Printf("Moving %s to sprint %d\n", issueKey, activeSprintId)
	_, err := j.Client.Sprint.MoveIssuesToSprint(activeSprintId, []string{issueKey})
	return err
}

