package main

import (
	"github.com/barakb/xap-trello"
	"fmt"
	"time"
	"regexp"
	"strconv"
)

const DATE = "2006-01-02"

func main() {
	//err := run()
	//if err != nil {
	//	fmt.Printf("Got error: %q, %#v\n", err.Error(), err)
	//} else {
	//	fmt.Println("All done")
	//}
	start, end, name, err := getNextSprintDefaults()
	fmt.Printf("start %s, end %s, name %q, err: %v", start.Format(DATE), end.Format(DATE), name, err)
}

func run() error {
	start, end, name, err := getNextSprintDefaults()
	if err != nil{
		return err
	}
	xapOpenJira, err := xap_trello.CreateXAPJiraOpen()
	if err != nil {
		return err
	}
	if xapOpenJira.ActiveSprint.Name != "" {
		fmt.Printf("Closing old sprint %s\n", xapOpenJira.ActiveSprint.Name)
		_, _, err = xapOpenJira.Client.Board.CloseSprint(fmt.Sprintf("%d", xapOpenJira.ActiveSprint.ID))
		if err != nil {
			return err
		}
	}

	//start, err := time.Parse(DATE, "2016-12-04")
	//if err != nil {
	//	return err
	//}
	//end, err := time.Parse(DATE, "2016-12-08")
	//if err != nil {
	//	return err
	//}

	fmt.Printf("Creating a new sprint %s\n", name)
	sprint, _, err := xapOpenJira.Client.Board.CreateSprint(name, start, end, xapOpenJira.MainScrumBoardId)
	if err != nil {
		return err
	}
	fmt.Printf("Moving items from trello to sprint %s\n", sprint.Name)
	err = xap_trello.Trello2Jira(3, sprint.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Starting sprint %s\n", sprint.Name)
	sprint, _, err = xapOpenJira.Client.Board.StartSprint(fmt.Sprintf("%d", sprint.ID))
	if err != nil {
		return err
	}
	return nil

	xapTrello, err := xap_trello.CreateXAPTrello()
	if err != nil {
		return err
	}
	board, err := xapTrello.Board("Personal")
	if err != nil {
		return err
	}
	lists, err := board.Lists()
	if err != nil {
		return err
	}
	for index, l := range lists {
		fmt.Printf("index %d, list:%s\n", index, l.Name)
		//if index == 0 {
		//	err := l.Close()
		//	return err
		//}
	}
	return board.AddList("New List", 0)

}

func getNextSprintDefaults() (start, end time.Time, name string, err error) {
	xapOpenJira, err := xap_trello.CreateXAPJiraOpen()
	if err != nil {
		return
	}
	sprint, _, err := xapOpenJira.Client.Board.GetLastSprint(fmt.Sprintf("%d", xapOpenJira.MainScrumBoardId))
	start = sprint.StartDate.AddDate(0, 0, 7)
	end = sprint.EndDate.AddDate(0, 0, 7)
	name, err = suggestNextSprintName(sprint.Name)
	fmt.Printf("Sprint is %+v\n", sprint)
	return
}

func suggestNextSprintName(prev string) (string, error) {
 	milestonePattern := regexp.MustCompile(`(?i)(.*)-M([0-9]+)`)
	match := milestonePattern.FindStringSubmatch(prev)
	if match != nil{
		i, err := strconv.Atoi(match[2])
		if err != nil{
			return "", fmt.Errorf("Failed to convert %q to int, name is %q\n", match[2], prev)
		}
		return fmt.Sprintf("%s-M%d", match[1], i + 1), nil
	}
 	rcPattern := regexp.MustCompile(`(?i)(.*)-rc([0-9]+)`)
	match = rcPattern.FindStringSubmatch(prev)
	if match != nil{
		i, err := strconv.Atoi(match[2])
		if err != nil{
			return "", fmt.Errorf("Failed to convert %q to int, name is %q\n", match[2], prev)
		}
		return fmt.Sprintf("%s-rc%d", match[1], i + 1), nil
	}
	return prev, nil
}