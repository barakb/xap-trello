package main

import (
	"fmt"
	"os"
	"github.com/barakb/go-jira"
)

func main() {
	fmt.Println("Jira")
	jiraClient, err := jira.NewClient(nil, "https://xap-issues.atlassian.net")
	if err != nil {
		panic(err)
	}
	fmt.Printf("jiraClient %v\n", jiraClient)
	res, err := jiraClient.Authentication.AcquireSessionCookie(os.Getenv("JIRA_USER"), os.Getenv("JIRA_PASSWORD"))
	if err != nil || res == false {
		fmt.Printf("Result: %v\n", res)
		panic(err)
	}

	issue, _, err := jiraClient.Issue.Get("XAP-12797")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Issue %v#\n", issue)
	fmt.Printf("Fields %v#\n", *issue.Fields)
	fmt.Printf("\n\n**************\nSummary:\n%s\n", issue.Fields.Summary)
	fmt.Printf("\n\n***************\nDescription:\n%s\n", issue.Fields.Description)


}
