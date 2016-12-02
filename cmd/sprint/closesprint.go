package main

import (
	"github.com/barakb/xap-trello"
	"fmt"
	"time"
)

func main() {
	err := run()
	if err != nil {
		fmt.Printf("Got error: %q, %#v\n", err.Error(), err)
	} else {
		fmt.Println("All done")
	}
}

func run() error {
	const DATE = "2006-01-02"
	const SPRINT_NAME = "12.1-M8"
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

	start, err := time.Parse(DATE, "2016-12-04")
	if err != nil {
		return err
	}
	end, err := time.Parse(DATE, "2016-12-08")
	if err != nil {
		return err
	}

	fmt.Printf("Creating a new sprint %s\n", SPRINT_NAME)
	sprint, _, err := xapOpenJira.Client.Board.CreateSprint(SPRINT_NAME, start, end, xapOpenJira.MainScrumBoardId)
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
}