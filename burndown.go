package xap_trello

import (
	"github.com/barakb/go-trello"
	"time"
	"regexp"
	"strconv"
	"log"
	"golang.org/x/net/context"
	"sync"
)

type TimelineEvent struct {
	Time     time.Time `json:"time"`
	Points   int `json:"points"`
	Day      string `json:"day"`
	CardId   string `json:"-"`
	CardName string `json:"-"`
}

type ListEvent struct {
	Time                   time.Time
	Id, Type, CardId, Name string
	Points                 int
}

func (l ListLog) Timeline() (timeline []TimelineEvent) {
	storyPoints := 0
	presenceMap := map[string]bool{}
	for _, event := range l.ListEvents {
		//log.Printf("Processing event %+v\n", event)
		if event.Points <= 0 {
			//log.Printf("skiping event %+v\n", event)
			continue
		}
		if event.Type == "create" || event.Type == "add" {
			if _, ok := presenceMap[event.CardId]; !ok {
				//log.Printf("Adding %d points because of %+v\n", event.Points, event)
				storyPoints += event.Points
				presenceMap[event.CardId] = true
			} else {
				continue
			}
		} else if _, ok := presenceMap[event.CardId]; ok {
			//log.Printf("removing %d points because of %+v\n", event.Points, event)
			storyPoints -= event.Points
			delete(presenceMap, event.CardId)
		} else {
			continue
		}
		day := toDayStr(event.Time)
		timeline = append(timeline, TimelineEvent{Time:event.Time, Points:storyPoints, Day:day, CardId:event.CardId, CardName:event.Name})
	}
	return timeline
}

func toDayStr(time time.Time) string{
	const layout = "Jan 2"
	return time.Format(layout)
}

type ListLog struct {
	TrelloClient  *Trello
	ListId        string
	ListEvents    []ListEvent
	LastEventId   string
	LastEventTime time.Time
}

func (l *ListLog)  UpdateFromTrello() error {
	lst, err := l.TrelloClient.Client.List(l.ListId)
	if err != nil {
		return err
	}
	actions, err := lst.Actions()
	if err != nil {
		return err
	}
	var lastId = ""
	var lastEventTime = ""
	if l.LastEventId != "" {
		index := findLastIdIndex(actions, l.LastEventId)
		actions = actions[0:index]
	}
	if 0 < len(actions) {
		lastId = actions[0].Id
		lastEventTime = actions[0].Date
	}
	//log.Printf("Reading %d events from trello\n", len(actions))
	for _, action := range reverseActions(actions) {
		// skip actions from the past
		if action.Type == "updateCard" {
			//log.Printf("**** Update card action%+v\n", action)
			if action.Data.ListBefore != action.Data.ListAfter {
				if lastId == "" {
					lastId = action.Id
					lastEventTime = action.Date
				}
				if action.Data.ListAfter.Id == lst.Id {
					l.AddEvent(action, "add")
					if err != nil {
						return err
					}
				} else if action.Data.ListBefore.Id == lst.Id {
					l.AddEvent(action, "rm")
					if err != nil {
						return err
					}
				}
			}
		} else if action.Type == "createCard" {
			if lastId == "" {
				lastId = action.Id
				lastEventTime = action.Date
			}
			l.AddEvent(action, "create")
			if err != nil {
				return err
			}
		}
	}
	if lastId != "" {
		//log.Printf("Updating lastId from %s to %s\n", l.LastEventId, lastId)
		l.LastEventId = lastId
		l.LastEventTime, _ = time.Parse(time.RFC3339, lastEventTime)
	}

	actionsView := replay(l.ListEvents)
	cards, err := lst.Cards()
	if err != nil {
		return err
	}
	// add all cards that their add event was lost.
	cardsMap := l.addMissingCardsEvents(cards, actionsView)
	// remove all cards that their remove event was lost
	cardsMap = l.removeExtraCardsEvents(actionsView, cardsMap, cards)
	//for key, _ := range actionsView {
	//	log.Printf("List %s contains card %s\n", lst.Name, key)
	//}
	return nil
}

func findLastIdIndex(actions []trello.Action, id string) int {
	for index, action := range actions {
		if action.Id == id {
			return index
		}
	}
	return len(actions)
}

func (l *ListLog)  AddEvent(action trello.Action, eventType string) error {
	listAction, err := fromTrelloAction(action, eventType)
	if err != nil {
		return err
	}
	l.ListEvents = append(l.ListEvents, *listAction)
	return nil
}


// remove all cards that their rm event was lost.
func (l *ListLog) removeExtraCardsEvents(actionsView map[string]bool, cardsMap map[string]*trello.Card, cards []trello.Card) (map[string]*trello.Card) {
	for _, card := range cards {
		cardsMap[card.Id] = &card
		if _, ok := actionsView[card.Id]; !ok {
			//log.Printf("-------------------- removeExtraCardsEvents missing Card Event !!!!!! %s\n", card.Name)
			l.ListEvents = append(l.ListEvents, ListEvent{Time:l.LastEventTime.Add(1 * time.Millisecond), Id:"", Type:"rm", CardId:card.Id, Name:card.Name, Points: points(card.Name)})
			actionsView[card.Id] = true
		}
	}
	return cardsMap
}

// add all cards that their add event was lost.
func (l *ListLog) addMissingCardsEvents(cards []trello.Card, actionsView map[string]bool) (map[string]*trello.Card) {
	cardsMap := map[string]*trello.Card{}
	for _, card := range cards {
		cardsMap[card.Id] = &card
		if _, ok := actionsView[card.Id]; !ok {
			//log.Printf("-------------------- addMissingCardsEvents missing Card Event !!!!!! %s\n", card.Name)
			l.ListEvents = append(l.ListEvents, ListEvent{Time:l.LastEventTime.Add(1 * time.Millisecond), Id:"", Type:"add", CardId:card.Id, Name:card.Name, Points: points(card.Name)})
			actionsView[card.Id] = true
		}
	}
	return cardsMap
}

func replay(events []ListEvent) map[string]bool {
	res := map[string]bool{}
	for _, action := range events {
		//log.Printf("play:%s %q, %d, at %s\n", action.Type, action.Name, action.Points, action.Time)
		if action.Type == "create" || action.Type == "add" {
			res[action.CardId] = true
		} else {
			delete(res, action.CardId)
		}
	}
	return res
}

func reverseActions(items []trello.Action) []trello.Action {
	res := items[:]
	for i, j := 0, len(res) - 1; i < j; i, j = i + 1, j - 1 {
		res[i], res[j] = res[j], res[i]
	}
	return res
}

func fromTrelloAction(action trello.Action, Type string) (*ListEvent, error) {
	t, err := time.Parse(time.RFC3339, action.Date)
	if err != nil {
		return nil, err
	}
	name := action.Data.Card.Name
	var points = points(name)
	return &ListEvent{Time: t, Id:action.Id, Type:Type, CardId:action.Data.Card.Id, Name: name, Points:points}, nil
}

func points(name string) (points int) {
	re := regexp.MustCompile("\\(([0-9]+)\\)")
	match := re.FindStringSubmatch(name)
	if len(match) == 2 {
		points, _ = strconv.Atoi(match[1])
	}
	return points
}

type ListWatcher struct {
	CancelFunc          context.CancelFunc
	currentTimelineLock sync.RWMutex
	currentTimeline     []TimelineEvent
}

func (lw ListWatcher) Timeline() []TimelineEvent {
	lw.currentTimelineLock.RLock()
	defer lw.currentTimelineLock.RUnlock()
	return lw.currentTimeline
}

func NewWatcher(selector func(trello.List) bool, waitDuration time.Duration) *ListWatcher {
	res := &ListWatcher{}
	ctx, cancel := context.WithCancel(context.Background())
	res.CancelFunc = cancel
	xapTrello, err := CreateXAPTrello()
	if err != nil {
		log.Fatal(err)
	}

	board, err := xapTrello.Board("XAP Scrum")
	if err != nil {
		log.Fatal(err)
	}

	trelloLists, err := board.Lists()
	if err != nil {
		log.Fatal(err)
	}

	ll := &ListLog{TrelloClient:xapTrello, ListId:"", ListEvents:nil}
	//todo on listId change start with fresh log.

	var timelineSize = 0;
	go func() {
		for {

			select {
			case <-ctx.Done():
				return
			default:
				for _, aList := range trelloLists {
					if selector(aList) {
						ll.ListId = aList.Id
						ll.UpdateFromTrello()
						timeline := ll.Timeline()
						//log.Printf("timeline (%d)\n", len(timeline))
						if timelineSize < len(timeline) {
							res.currentTimelineLock.Lock()
							res.currentTimeline = timeline
							res.currentTimelineLock.Unlock()
							for _, timelineEvent := range timeline {
								log.Printf("points:%d, card:%q, time:%s\n", timelineEvent.Points, timelineEvent.CardName, timelineEvent.Time)
							}
							timelineSize = len(timeline)
							log.Println("\n\n\n***********************\n\n\n ")
						}
						time.Sleep(waitDuration)
					}
				}

			}

		}
	}()
	return res
}