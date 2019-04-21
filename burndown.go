package xap_trello

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/barakb/go-trello"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BurnDownData struct {
	TrelloEvents []TrelloState `json:"trello_events"`
	SprintStatus SprintStatus  `json:"sprint_status"`
	Version      int           `json:"version"`
	Sprint       *Sprint
}

type Burndown struct {
	BurnDownData
	sync.RWMutex
	done     chan struct{}
	commands chan BurndownCommand
	Trello   *Trello
}

type Sprint struct {
	Name  string    `json:"name"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func readSprint(path string) (*Sprint, error) {
	fileHandler, err := os.Open(path)
	if err != nil {
		log.Printf("error %s\n", err.Error())
		return nil, err
	}
	defer fileHandler.Close()
	reader := bufio.NewReader(fileHandler)
	sprint := Sprint{}
	if err := json.NewDecoder(reader).Decode(&sprint); err != nil {
		log.Printf("error %s\n", err.Error())
		return nil, err
	}
	return &sprint, nil
}

func writeSprint(sprint Sprint, path string) error {
	fileHandler, err := os.Create(path)
	if err != nil {
		log.Printf("error %s\n", err.Error())
		return err
	}

	defer fileHandler.Close()
	writer := bufio.NewWriter(fileHandler)
	defer writer.Flush()
	if err := json.NewEncoder(writer).Encode(&sprint); err != nil {
		log.Printf("error %s\n", err.Error())
		return err
	}
	return nil
}

func NewBurnDown() *Burndown {

	xapTrello, err := CreateXAPTrello()
	if err != nil {
		log.Fatal(err)
	}
	burndown := &Burndown{Trello: xapTrello, commands: make(chan BurndownCommand), BurnDownData: BurnDownData{Sprint: ReadSprint()}}
	go burndown.ScanLoop(10 * time.Second) //todo remove
	return burndown
}

func ReadSprint() *Sprint {
	s, e := readSprint("sprint.json")
	if e != nil {
		log.Printf("error while reading sprint from file: %s\n", e.Error())
		return nil
	}
	return s
}

func WriteSprint(sprint Sprint) error {
	return writeSprint(sprint, "sprint.json")
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
	Name       string      `json:"name"`
	Total      interface{} `json:"total"`
	Bottom     interface{} `json:"bottom"`
	Top        interface{} `json:"top"`
	Expected   float64     `json:"expected"`
	WorkingDay bool        `json:"working_day"`
}

type SprintStatus struct {
	Version int
	Name    string `json:"name"`
	Days    []Day  `json:"days"`
	Today   int    `json:"today"`
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
	date := b.Sprint.Start
	for date.Before(b.Sprint.End) {
		dayStr := toDayStr(date)
		order = append(order, dayStr)
		m[dayStr] = true
		nextDay := date.Add(24 * time.Hour)
		date = nextDay
	}
	dayStr := toDayStr(b.Sprint.End)
	order = append(order, dayStr)
	m[dayStr] = true

	s = &SprintStatus{Name: b.Sprint.Name, Today: indexOf(order, toDayStr(time.Now())), Days: []Day{}}
	firstDay, err := b.findFirstFilledDay()
	if err != nil {
		if 0 < len(b.TrelloEvents) {
			firstDay = b.TrelloEvents[len(b.TrelloEvents)-1]
		} else {
			log.Printf("Fail to find data, error is: %q\n", err)
			return s
		}
	}
	total := firstDay.Done + firstDay.InProgress + firstDay.Planned
	planningTotal := total
	perDay := float64(total) / float64(len(order))
	pointsAdded := 0
	var lastDay *Day
	for index, name := range order {
		expected := float64(total) - (float64(index+1) * perDay)
		day := &Day{Name: name, Expected: expected, WorkingDay: true}
		if e, ok := timeline[name]; ok && index <= s.Today {
			day.Total = e.Done + e.Planned + e.InProgress
			total = day.Total.(int)
			day.Expected = float64(total) - (float64(index+1) * perDay)
			day.Top = day.Total.(int) - e.Done
			if lastDay != nil {
				pointsAdded += (day.Total.(int) - lastDay.Total.(int))
			}
			day.Bottom = pointsAdded
		} else {
			day.Total = total
			day.Bottom = pointsAdded
		}
		s.Days = append(s.Days, *day)
		lastDay = day

	}
	s.Days = append([]Day{{Name: "Planning", Top: planningTotal, WorkingDay: false, Expected: float64(planningTotal), Total: planningTotal, Bottom: 0}}, s.Days...)
	return s
}

func (b *Burndown) linearizedEvents() []TrelloState {
	res := []TrelloState{}
	var last *TrelloState
	var lastDay = ""
	for index, event := range b.TrelloEvents {
		current := &event
		day := toDayStr(current.Time)
		if lastDay == "" {
			last = current
			lastDay = day
			continue
		}
		if lastDay != day {
			res = append(res, *last)
			last = current
			lastDay = day
		} else if index == len(b.TrelloEvents)-1 {
			res = append(res, *current)
		}
	}
	return res
}

func (b *Burndown) findFirstFilledDay() (TrelloState, error) {
	lin := b.linearizedEvents()
	for _, event := range lin {
		if event.InProgress != 0 || event.Planned != 0 || event.Done != 0 {
			return event, nil
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

		if len(b.TrelloEvents) == 0 || !sprintState.sameAs(b.TrelloEvents[len(b.TrelloEvents)-1]) {
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
	startDate := b.Sprint.Start
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
func (b *Burndown) commitAndPush() error {
	startDate := b.Sprint.Start
	filename := fmt.Sprintf("%d-%02d-%02d-%s-logs.json", startDate.Year(), startDate.Month(), startDate.Day(), b.Sprint.Name)
	token, err := ReadGithubToken()
	if err != nil {
		return err
	}
	git := NewGitRepository("data", "https://github.com/barakb/imc-sprints.git", token.AccessToken)
	git.path = "/usr/local/git/bin/"
	err = git.Init()
	if err != nil {
		return err
	}

	err = git.Add(filename)
	if err != nil {
		return err
	}

	err = git.Commit(fmt.Sprintf("Automatic update of file %s on %v", filename, time.Now()))
	if err != nil {
		return err
	}

	err = git.Rebase()
	if err != nil {
		return err
	}

	err = git.Push()
	if err != nil {
		return err
	}
	return nil
}

func (b *Burndown) load() (err error) {
	startDate := b.Sprint.Start

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
	err := b.save()
	if err != nil {
		fmt.Printf("Got error %s while trying to sage changes\n", err.Error())
	}
	err = b.commitAndPush()
	if err != nil {
		fmt.Printf("Got error %s while trying to commit changes\n", err.Error())
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
			if err != nil {
				return err
			}
			break
		}
	}

	err = board.AddList(fmt.Sprintf("Done in %s", name), 0)
	if err != nil {
		return err
	}

	b.Sprint = ReadSprint()
	b.SprintStatus = SprintStatus{}
	b.TrelloEvents = []TrelloState{}
	b.Version = 0
	return nil
}

func sumPoints(lst trello.List) (p int) {
	cards, _ := lst.Cards()
	for _, card := range cards {
		p += points(card.Name)
	}
	return p
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
