package main

import (
	"fmt"
	"log"
	"github.com/VojtechVitek/go-trello"
	"github.com/vharitonsky/iniflags"
	"time"
	"sort"
	"regexp"
	"strconv"
	"sync"
	"flag"
	"net/http"
	"encoding/json"
	"strings"
)

var port = flag.Int("port", 8080, "Configure the server port")

var appKey = flag.String("appKey", "6c4a296ac4e1e976ff7ef59aa7719e75", "the trello app key, see https://trello.com/app-key ")
var appToken = flag.String("appToken", "", "the trello app token, see https://trello.com/app-key")
var api *trello.Client

func main() {
	iniflags.Parse()
	var err error;
	api, err = trello.NewAuthClient(*appKey, appToken)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", hello)
	fmt.Printf("Server started on https://localhost:%d\n", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	fmt.Printf("listenAndServe returns %v\n", err)
}

func hello(w http.ResponseWriter, r *http.Request) {
	//io.WriteString(w, "Hello world!")
	//log.Printf("request url is %s\n", r.RequestURI)
	if r.RequestURI == "/favicon.ico" {
		http.Error(w, "no favicon yet", http.StatusNotFound)
		return;
	}
	burndown := getBurndownChart("12.1.0-m1", api)
	bytes, err := json.Marshal(burndown)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(bytes)
	}
}

type Event struct {
	Date          string   `json:"date"`
	RemainsCards  int     `json:"remainsCards"`
	RemainsPoints int     `json:"remainsPoints"`
	Name          string     `json:"cardName"`
	Url           string     `json:"cardUrl"`
}

type Burndown struct {
	StartDate   string   `json:"startDate"`
	EndDate     string   `json:"endDate"`
	Date        string   `json:"date"`
	TotalCards  int     `json:"totalCards"`
	TotalPoints int     `json:"totalPoints"`
	Timeline    []Event    `json:"timeline"`
}

type cardWrapper struct {
	card     *trello.Card
	points   int
	doneTime time.Time
}

type SortedDoneCards []cardWrapper

type SprintMetadata struct {
	name                 string
	startDate, endDate   time.Time
	datesToExclude       [] time.Time
	estimatedStoryPoints int
}

func getCardActions(card *trello.Card) []trello.Action {
	res, err := card.Actions()
	if (err != nil) {
		log.Fatal(err)
	}
	return res;
}
func getDoneTime(card trello.Card) time.Time {
	actions := getCardActions(&card)
	for _, action := range actions {
		//fmt.Printf("action type: %s, action date %v, action data.ListBefore: %v, action data.ListAfter: %v,  board name: %v\n", action.Type, action.Date, action.Data.ListBefore, action.Data.ListAfter, action.Data.Board.Name)
		if action.Type == "updateCard" && action.Data.ListBefore.Name != "Done" && action.Data.ListAfter.Name == "Done" {
			t, err := time.Parse(time.RFC3339, action.Date)
			if err != nil {
				log.Fatal(err)
			}
			return t
		}
	}
	return time.Time{}
}

func (a SortedDoneCards) Len() int {
	return len(a)
}
func (a SortedDoneCards) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a SortedDoneCards) Less(i, j int) bool {
	return a[i].doneTime.Before(a[j].doneTime)
}

func handleDoneList(start, end  time.Time, lst trello.List) chan []cardWrapper {
	doneCards := []cardWrapper{}
	res := make(chan []cardWrapper, 1)
	cardsWrapperChan := make(chan cardWrapper, 30)
	cards, _ := lst.Cards()
	var wg sync.WaitGroup
	wg.Add(len(cards))
	for _, card := range cards {
		go func(card trello.Card) {
			defer wg.Done()
			t := getDoneTime(card)
			//fmt.Printf("Done time is %v\n", t)
			if t.Before(start) || t.After(end) {
				fmt.Printf("Skiping card %s it's done date %v is not in this sprint (%v - %v)\n", card.Name, t, start, end)
			} else {
				cardsWrapperChan <- cardWrapper{&card, getCardPoints(card), t}
			}
		}(card)
	}
	go func() {
		wg.Wait()
		close(cardsWrapperChan)
		for cw := range cardsWrapperChan {
			doneCards = append(doneCards, cw)
		}
		res <- doneCards
		close(res)
	}()

	return res
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

func findList(lists []trello.List, name string) (trello.List, error) {
	for _, list := range lists {
		if list.Name == name {
			return list, nil
		}
	}
	return trello.List{}, fmt.Errorf("Failed to find list %q\n", name)
}

func getBurndownChart(sprintName string, api *trello.Client) Burndown {
	burndown := Burndown{}
	burndown.Date = time.Now().Format(time.RFC3339)

	var doneCardsChan  chan []cardWrapper
	doneCardsSlice := []cardWrapper{}
	sprintCardsChan := make(chan int, 100)

	u, err := api.Member("barakbarorion")
	if err != nil {
		log.Fatal(err)
	}

	board, err := getBoard("XAP Scrum", u)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("* %v (%v)  ----- \n", board.Name, board.ShortUrl)

	lists, err := board.Lists()
	if err != nil {
		log.Fatal(err)
	}
	sprintMetadata, err := findSprintMetadata(lists, "")
	if err != nil {
		log.Fatal(err)
	}
	burndown.StartDate = sprintMetadata.startDate.Format(time.RFC3339)
	burndown.EndDate = sprintMetadata.endDate.Format(time.RFC3339)

	log.Printf("sprintMetadata: %#v\n", sprintMetadata)

	sprint, err := findList(lists, sprintMetadata.name)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	doneCardsChan = handleDoneList(sprintMetadata.startDate, sprintMetadata.endDate, sprint)
	if sprintName == "" {
		inProgress, err := findList(lists, "In Progress")
		if err != nil {
			log.Fatal(err)
		}
		planned, err := findList(lists, "Planned")
		if err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			cards, _ := inProgress.Cards()
			for _, card := range cards {
				//fmt.Printf("Adding spring card card %s from list %s\n", card.Name, lst.Name)
				sprintCardsChan <- getCardPoints(card)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			cards, _ := planned.Cards()
			for _, card := range cards {
				//fmt.Printf("Adding spring card card %s from list %s\n", card.Name, lst.Name)
				sprintCardsChan <- getCardPoints(card)
			}
		}()

	}

	wg.Wait()
	close(sprintCardsChan)
	doneCardsSlice = <-doneCardsChan
	//fmt.Println("Sorting")
	sort.Sort(SortedDoneCards(doneCardsSlice))
	totalPoints := 0;
	totalCards := 0;
	donePoints := 0;
	doneCards := 0;
	for _, dc := range doneCardsSlice {
		//fmt.Printf("done %v card %s\n", dc.doneTime, dc.card.Name)
		doneCards += 1
		donePoints += dc.points
		totalCards += 1
		totalPoints += dc.points
	}
	for sc := range sprintCardsChan {
		//fmt.Printf("sprint card worth %d points\n", sc)
		totalCards += 1
		totalPoints += sc
	}
	burndown.TotalCards = totalCards
	burndown.TotalPoints = totalPoints
	remainsCards := totalCards;
	remainsPoints := totalPoints;
	for _, dc := range doneCardsSlice {
		remainsCards -= 1
		remainsPoints -= dc.points
		event := Event{}
		event.Date = dc.doneTime.Format(time.RFC3339)
		event.RemainsCards = remainsCards
		event.RemainsPoints = remainsPoints
		event.Name = dc.card.Name
		event.Url = dc.card.Url
		burndown.Timeline = append(burndown.Timeline, event)
	}

	return burndown
}

func findSprintMetadata(lists []trello.List, sprintName string) (SprintMetadata, error) {
	const SPRINTS = "Sprints"
	for _, lst := range lists {
		if lst.Name == SPRINTS {
			sprint, err := findCard(lst, sprintName)
			if err != nil {
				return SprintMetadata{}, err

			}
			return parseSprint(sprint)
		}
	}
	return SprintMetadata{}, fmt.Errorf("Failed to find list named %q\n", SPRINTS)
}

func parseSprint(card trello.Card) (res SprintMetadata, err error) {
	if len(card.Desc) < 2 {
		return res, fmt.Errorf("Failed to parse spring card %q, desc is %q\n", card.Name, card.Desc)
	}
	res.name = card.Name
	lines := strings.Split(card.Desc, "\n")
	re := regexp.MustCompile("^\\s*([^:]+)\\s*:\\s*([^:]+)\\s*$")
	for _, line := range lines {
		result := re.FindStringSubmatch(line)
		if len(result) < 3 {
			return res, fmt.Errorf("Failed to parse spring card %q, desc is %q, offending line is: %q\n", card.Name, card.Desc, line)
		}
		attr := result[1]
		value := result[2]
		if strings.EqualFold("startDate", attr) {
			res.startDate, err = time.Parse("2006-01-02", value)
		} else if strings.EqualFold("endDate", attr) && err == nil {
			res.endDate, err = time.Parse("2006-01-02", value)
		} else if strings.EqualFold("datesToExclude", attr) && err == nil {
			res.datesToExclude = parseListOfDates(value)
		} else if strings.EqualFold("estimatedStoryPoints", attr) && err == nil {
			res.estimatedStoryPoints, err = strconv.Atoi(value)
		}
		//log.Printf("attr is: %q, value is %q, res is %v\n", attr, value, res)

	}
	log.Printf("returning res %v\n", res);
	return res, err
}
func parseListOfDates(str string) (res [] time.Time) {
	dates := strings.Split(str, ",")
	for _, dateStr := range dates {
		dateStr = strings.TrimSpace(dateStr)
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			res = append(res, date)
		}
	}
	return res
}

func findCard(list trello.List, sprintName string) (trello.Card, error) {
	cards, err := list.Cards()
	if err != nil {
		return trello.Card{}, err
	}
	if sprintName == "" {
		if len(cards) == 0 {
			return trello.Card{}, fmt.Errorf("Failed to find last sprint metadata in list %q, list is empty\n", list.Name)
		}
		return cards[len(cards) - 1], nil
	} else {
		for _, card := range cards {
			if card.Name == sprintName {
				return card, nil
			}
		}
	}
	return trello.Card{}, fmt.Errorf("Failed to find sprint %q metadata in list %q\n", sprintName, list.Name)
}

func getCardPoints(card trello.Card) int {
	r := regexp.MustCompile(`^.*\(([0-9]+)\).*$`)
	result := r.FindStringSubmatch(card.Name)
	if len(result) < 2 {
		return 1
	}
	i, err := strconv.Atoi(result[1])
	if err == nil {
		return i
	}
	return 1
}