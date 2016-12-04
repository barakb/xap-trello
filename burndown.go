package xap_trello

import (
	"github.com/barakb/go-trello"
	"log"
	"time"
	"github.com/barakb/go-jira"
	"os"
	"fmt"
	"bufio"
	"encoding/json"
	"io/ioutil"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type BurnDownData struct {
	TrelloEvents []TrelloState `json:"trello_events"`
	SprintStatus SprintStatus `json:"sprint_status"`
	Version      int `json:"version"`
	Sprint   jira.Sprint
}

type Burndown struct {
	BurnDownData
	sync.RWMutex
	done     chan struct{}
	commands chan BurndownCommand
	Trello   *Trello
	Jira     *Jira
}

func NewBurnDown() *Burndown {
	xapOpenJira, err := CreateXAPJiraOpen()
	if err != nil {
		log.Fatal(err)
	}

	xapTrello, err := CreateXAPTrello()
	if err != nil {
		log.Fatal(err)
	}
	burndown := &Burndown{Trello:xapTrello, commands: make(chan BurndownCommand), Jira:xapOpenJira, BurnDownData:BurnDownData{Sprint:xapOpenJira.ActiveSprint}}
	go burndown.ScanLoop(10 * time.Second) //todo remove
	return burndown
}

type BurndownCommand func(burndown *Burndown)

type TrelloState struct {
	Done, InProgress, Planned int
	Time                      time.Time
}

func (ss TrelloState) sameAs(other TrelloState) bool {
	return ss.Planned == other.Planned && ss.InProgress == other.InProgress && ss.Done == other.Done && toDayStr(ss.Time) == toDayStr(other.Time)
}

type Day struct {
	Name       string `json:"name"`
	Total      interface{} `json:"total"`
	Bottom     interface{} `json:"bottom"`
	Top        interface{} `json:"top"`
	Expected   float64 `json:"expected"`
	WorkingDay bool `json:"working_day"`
}

type SprintStatus struct {
	Version int
	Name    string `json:"name"`
	Days    []Day `json:"days"`
	Today   int `json:"today"`
}

func (b *Burndown) statePerDay(events []TrelloState) map[string]TrelloState {
	m := map[string]TrelloState{}
	for _, event := range events {
		eventDay := toDayStr(event.Time)
		m[eventDay] = event
	}
	return m
}

func (b *Burndown) createSprint(timeline map[string]TrelloState) (s *SprintStatus) {
	order := []string{}
	m := map[string]bool{}
	date := b.Sprint.StartDate
	for date.Before(*b.Sprint.EndDate) {
		dayStr := toDayStr(*date)
		order = append(order, dayStr)
		m[dayStr] = true
		nextDay := date.Add(24 * time.Hour)
		date = &nextDay
	}
	dayStr := toDayStr(*b.Sprint.EndDate)
	order = append(order, dayStr)
	m[dayStr] = true

	s = &SprintStatus{Name:b.Sprint.Name, Today:indexOf(order, toDayStr(time.Now())), Days:[]Day{}}
	firstDay, err := findFirstFilledDay(timeline, order)
	if err != nil {
		if 0 < len(b.TrelloEvents){
			firstDay = b.TrelloEvents[len(b.TrelloEvents) - 1]
		}else {
			log.Printf("Fail to find data, error is: %q\n", err)
			return s
		}
	}
	total := firstDay.Done + firstDay.InProgress + firstDay.Planned
	perDay := float64(total) / float64(len(order))
	pointsAdded := 0
	var lastDay *Day
	for index, name := range order {
		expected := float64(total) - (float64(index + 1) * perDay)
		day := &Day{Name:name, Expected:expected, WorkingDay:true}
		if e, ok := timeline[name]; ok &&  index <= s.Today {
			day.Total = e.Done + e.Planned + e.InProgress
			day.Top = day.Total.(int) - e.Done
			if lastDay != nil {
				log.Printf("Added points lastDay is: %+v\n", lastDay)
				pointsAdded += (day.Total.(int) - lastDay.Total.(int))
			}
			day.Bottom = pointsAdded
		} else {
			day.Total = total
			day.Bottom = pointsAdded
		}
		log.Printf("Processed day %+v\n", day)
		s.Days = append(s.Days, *day)
		lastDay = day

	}
	s.Days = append([]Day{{Name:"Planning", Top:total, WorkingDay:false, Expected:float64(total), Total:total, Bottom:0}}, s.Days...)
	return s
}

func findFirstFilledDay(timeline map[string]TrelloState, order []string) (TrelloState, error) {
	for _, day := range order {
		state := timeline[day]
		if state.InProgress != 0 || state.Planned != 0 || state.Done != 0 {
			return state, nil
		}
	}
	return TrelloState{}, errors.New("Not found")
}

func (b *Burndown) scanOnce() (res TrelloState, err error) {
	log.Println("ScanOnce")
	board, err := b.Trello.Board("XAP Scrum")
	if err != nil {
		return res, err
	}

	trelloLists, err := board.Lists()
	if err != nil {
		return res, err
	}

	for index, trelloList := range trelloLists {
		if index == 0 {
			res.Done = sumPoints(trelloList)
		} else if index == 1 {
			res.InProgress = sumPoints(trelloList)
		} else if index == 2 {
			res.Planned = sumPoints(trelloList)
		} else {
			break
		}
	}
	res.Time = time.Now()
	return res, nil
}

func (b *Burndown) ScanLoop(delay time.Duration) {
	b.load()
	compressedTimeline := b.compressTimeline()
	sprintStatus := b.createSprint(compressedTimeline)
	sprintStatus.Version = b.Version
	b.Version = b.Version + 1
	b.RWMutex.Lock()
	b.SprintStatus = *sprintStatus
	b.RWMutex.Unlock()
	for {
		sprintState, err := b.scanOnce()
		if err != nil {
			log.Fatalf("got error %q, while calling burndown.ScanOnce()", err.Error())
		}

		//log.Printf("sprintState is %+v\n", sprintState)

		if len(b.TrelloEvents) == 0 || !sprintState.sameAs(b.TrelloEvents[len(b.TrelloEvents) - 1]) {
			b.TrelloEvents = append(b.TrelloEvents, sprintState)
			log.Printf("Timeline changed %v\n", b.TrelloEvents)
			compressedTimeline := b.compressTimeline()
			sprintStatus := b.createSprint(compressedTimeline)
			sprintStatus.Version = b.Version
			b.Version = b.Version + 1
			b.RWMutex.Lock()
			b.SprintStatus = *sprintStatus
			b.RWMutex.Unlock()
			err := b.save()
			if err != nil {
				log.Printf("Error %q, while saving\n", err.Error())
			}
		}
		select {
		case <-b.done:
			log.Println("ScanLoop exiting")
			return
		case cmd := <-b.commands:
			cmd(b)
		case <-time.After(delay):
			continue
		}
	}
}

func (b *Burndown) GetSprintStatus() *SprintStatus {
	b.RWMutex.RLock()
	defer b.RWMutex.RUnlock()
	return &b.SprintStatus
}

func (b *Burndown) compressTimeline() map[string]TrelloState {
	m := map[string]TrelloState{}
	for _, event := range b.TrelloEvents {
		eventDay := toDayStr(event.Time)
		m[eventDay] = event
	}
	return m
}

func (b *Burndown) save() error {
	err := os.MkdirAll("data", os.ModePerm)
	if err != nil {
		return err
	}
	startDate := b.Sprint.StartDate
	filename := fmt.Sprintf("data/%d-%02d-%02d-%s-logs.json", startDate.Year(), startDate.Month(), startDate.Day(), b.Sprint.Name)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	bytes, err := json.MarshalIndent(b.BurnDownData, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(bytes)
	if err != nil {
		return err
	}
	w.Flush()
	return nil
}

func (b *Burndown) load() (err error) {
	startDate := b.Sprint.StartDate
	filename := fmt.Sprintf("data/%d-%02d-%02d-%s-logs.json", startDate.Year(), startDate.Month(), startDate.Day(), b.Sprint.Name)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, &(b.BurnDownData))
	if err != nil {
		return err
	}
	return nil
}

func (b *Burndown) StartNewSprint(name string, start, end time.Time) chan struct{} {
	res := make(chan struct{})
	go func() {
		b.commands <- func(b *Burndown) {
			b.startNewSprint(name, start, end)
			go func() {
				res <- struct{}{}
			}()
		}
	}()
	return res
}

func (b *Burndown) startNewSprint(name string, start, end time.Time) error {
	b.save()

	if b.Jira.ActiveSprint.Name != "" {
		fmt.Printf("Closing old sprint %s\n", b.Jira.ActiveSprint.Name)
		_, _, err := b.Jira.Client.Board.CloseSprint(fmt.Sprintf("%d", b.Jira.ActiveSprint.ID))
		if err != nil {
			return err
		}
	}

	board, err := b.Trello.Board("XAP Scrum")
	if err != nil {
		return err
	}
	lists, err := board.Lists()
	if err != nil {
		return err
	}
	for index, l := range lists {
		if index == 0 {
			err := l.Close()
			if err != nil{
				return err
			}
			break;
		}
	}

	err = board.AddList(fmt.Sprintf("Done in %s", name), 0)
	if err != nil {
		return err
	}

	fmt.Printf("Creating a new sprint %s\n", name)
	sprint, _, err := b.Jira.Client.Board.CreateSprint(name, start, end, b.Jira.MainScrumBoardId)
	if err != nil {
		return err
	}
	fmt.Printf("Moving items from trello to sprint %s\n", sprint.Name)
	err = Trello2Jira(3, sprint.ID)
	if err != nil {
		return err
	}
	fmt.Printf("Starting sprint %s\n", sprint.Name)
	sprint, _, err = b.Jira.Client.Board.StartSprint(fmt.Sprintf("%d", sprint.ID))
	if err != nil {
		return err
	}


	//todo change active sprint
	b.Sprint = *sprint
	b.SprintStatus = SprintStatus{}
	b.TrelloEvents = []TrelloState{}
	b.Version = 0
	b.Jira.ActiveSprint = *sprint
	return nil
}

func sumPoints(lst trello.List) (p int) {
	cards, _ := lst.Cards()
	for _, card := range cards {
		p += points(card.Name)
	}
	return p;
}

func toDayStr(time time.Time) string {
	const layout = "Mon, Jan 2"
	return time.Format(layout)
}

func points(name string) (points int) {
	re := regexp.MustCompile("\\(([0-9]+)\\)")
	match := re.FindStringSubmatch(name)
	if len(match) == 2 {
		points, _ = strconv.Atoi(match[1])
		return points
	}

	if strings.Contains(strings.ToUpper(name), "{S}") || strings.Contains(strings.ToUpper(name), "{SMALL}") {
		return 5
	}
	if strings.Contains(strings.ToUpper(name), "{M}") || strings.Contains(strings.ToUpper(name), "{MED}") {
		return 25
	}
	if strings.Contains(strings.ToUpper(name), "{L}") || strings.Contains(strings.ToUpper(name), "{LARGE}") {
		return 100
	}

	return points
}
