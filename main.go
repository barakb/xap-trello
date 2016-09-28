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
	burndown := getBurndownChart(time.Now().AddDate(0, 0, -3), time.Now().AddDate(0, 0, 3), api)
	bytes, err := json.Marshal(burndown)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(bytes)
	}
}

type Event struct {
	Date        string   `json:"date"`
	DoneCards   int     `json:"doneCards"`
	DonePoints  int     `json:"donePoints"`
}

type Burndown struct {
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	Date      string   `json:"date"`
	TotalCards  int     `json:"totalCards"`
	TotalPoints int     `json:"totalPoints"`
	Timeline  []Event    `json:"timeline"`
}

type cardWrapper struct {
	card     *trello.Card
	points   int
	doneTime time.Time
}

type SortedDoneCards []cardWrapper

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

func getBurndownChart(start, end  time.Time, api *trello.Client)  Burndown {
	burndown := Burndown{}
	burndown.StartDate = start.Format(time.RFC3339)
	burndown.EndDate = end.Format(time.RFC3339)
	burndown.Date = time.Now().Format(time.RFC3339)


	var doneCardsChan  chan []cardWrapper
	doneCardsSlice := []cardWrapper{}
	sprintCardsChan := make(chan int, 100)

	u, err := api.Member("barakbarorion")
	if err != nil {
		log.Fatal(err)
	}

	boards, err := u.Boards()
	if err != nil {
		log.Fatal(err)
	}

	for _, board := range boards {
		if board.Name == "XAP Scrum" {
			fmt.Printf("* %v (%v)  ----- \n", board.Name, board.ShortUrl)

			lists, err := board.Lists()
			if err != nil {
				log.Fatal(err)
			}
			var wg sync.WaitGroup
			for _, lst := range lists {
				if lst.Name == "Done" {
					doneCardsChan = handleDoneList(start, end, lst)
				} else if lst.Name == "in-progress" || lst.Name != "Current Sprint" {
					//fmt.Printf("processing list %s\n", lst.Name)
					wg.Add(1)
					go func(lst trello.List) {
						defer wg.Done()
						cards, _ := lst.Cards()
						for _, card := range cards {
							//fmt.Printf("Adding spring card card %s from list %s\n", card.Name, lst.Name)
							sprintCardsChan <- getCardPoints(card)
						}
					}(lst)
				}
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
			//fmt.Printf("doneCards: %v\n", doneCards)
			//fmt.Printf("totalCards: %d\n", totalCards)
			//fmt.Printf("donePoints: %d\n", donePoints)
			//fmt.Printf("totalPoints: %d\n", totalPoints)
			burndown.TotalCards = totalCards
			burndown.TotalPoints = totalPoints
			doneCards = 0;
			donePoints = 0;
			for _, dc := range doneCardsSlice {
				doneCards += 1
				donePoints += dc.points
				event := Event{}
				event.Date = dc.doneTime.Format(time.RFC3339)
				event.DoneCards = doneCards
				event.DonePoints = donePoints
				burndown.Timeline = append(burndown.Timeline, event)
			}

		}
	}
	return burndown
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