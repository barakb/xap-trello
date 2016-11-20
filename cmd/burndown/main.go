package main

import (
	"github.com/barakb/xap-trello"
	"log"
	"fmt"
	"time"
	"github.com/barakb/go-trello"
)

type ListEvent struct {
	Time             time.Time
	Id, Type, CardId string
}

func main() {
	xapTrello, err := xap_trello.CreateXAPTrello()
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
	for {
		for _, aList := range trelloLists {
			//if 3 <= n {
			//	break
			//}
			//var listActions []ListEvent
			if aList.Name == "Barak" {
				ll := ListLog{TrelloClient:xapTrello, ListId:aList.Id, ListEvents:nil}
				ll.UpdateFromTrello()
			}
			//computeTimeLine(aList)
		}
		time.Sleep(time.Second * 2)
	}
}

type ListLog struct {
	TrelloClient *xap_trello.Trello
	ListId       string
	ListEvents   []ListEvent
}

func (l ListLog)  UpdateFromTrello() error {
	lst, err := l.TrelloClient.Client.List(l.ListId)
	if err != nil {
		return err
	}
	actions, err := lst.Actions()
	if err != nil {
		return err
	}
	for _, action := range actions {
		//log.Printf("Action id:%s, type:%q, data: %+v\n",action.Id, action.Type, action.Data)
		if action.Type == "updateCard" {
			if action.Data.ListBefore != action.Data.ListAfter {
				if action.Data.ListAfter.Id == lst.Id {
					listAction, err := fromTrelloAction(action, "add")
					if err != nil {
						return err
					}
					l.ListEvents = append(l.ListEvents, *listAction)
					fmt.Printf("%+v\n", *listAction)
				} else {
					listAction, err := fromTrelloAction(action, "rm")
					if err != nil {
						return err
					}
					l.ListEvents = append(l.ListEvents, *listAction)
					fmt.Printf("%+v\n", *listAction)
				}
			}
		} else if action.Type == "createCard" {
			listAction, err := fromTrelloAction(action, "create")
			if err != nil {
				return err
			}
			l.ListEvents = append(l.ListEvents, *listAction)
			fmt.Printf("%+v\n", *listAction)
		}

	}
	actionsView := replay(l.ListEvents)
	cards, err := lst.Cards()
	if err != nil {
		return err
	}
	// add all cards that their add event was lost.
	listActions, cardsMap := addMissingCardsEvents(cards, actionsView, l.ListEvents)
	l.ListEvents = listActions
	// remove all cards that their remove event was lost
	listActions, cardsMap = removeExtraCardsEvents(actionsView, cardsMap, cards, l.ListEvents)
	l.ListEvents = listActions

	for key, _ := range actionsView {
		log.Printf("List %s contains card %s\n", lst.Name, key)
	}
	return nil
}



// remove all cards that their rm event was lost.
func removeExtraCardsEvents(actionsView map[string]bool, cardsMap map[string]*trello.Card, cards []trello.Card, actions []ListEvent) ([]ListEvent, map[string]*trello.Card) {
	for _, card := range cards {
		cardsMap[card.Id] = &card
		if _, ok := actionsView[card.Id]; !ok {
			actions = append(actions, ListEvent{Time:time.Now(), Id:"", Type:"add", CardId:card.Id})
			actionsView[card.Id] = true
		}
	}
	return actions, cardsMap
}

// add all cards that their add event was lost.
func addMissingCardsEvents(cards []trello.Card, actionsView map[string]bool, actions []ListEvent) ([]ListEvent, map[string]*trello.Card) {
	cardsMap := map[string]*trello.Card{}
	for _, card := range cards {
		cardsMap[card.Id] = &card
		if _, ok := actionsView[card.Id]; !ok {
			actions = append(actions, ListEvent{Time:time.Now(), Id:"", Type:"add", CardId:card.Id})
			actionsView[card.Id] = true
		}
	}
	return actions, cardsMap
}

func replay(actions []ListEvent) map[string]bool {
	reversed := reverse(actions)
	res := map[string]bool{}
	for _, action := range reversed {
		if action.Type == "createCard" || action.Type == "add" {
			res[action.CardId] = true
		} else {
			delete(res, action.CardId)
		}
	}
	return res
}

func reverse(actions []ListEvent) []ListEvent {
	res := actions[:]
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
	return &ListEvent{Time: t, Id:action.Id, Type:Type, CardId:action.Data.Card.Id}, nil
}

//type TimelineEvent struct {
//	Time time.Time
//	Card trello.Card
//	From, To string
//}
//
//func computeTimeLine(card trello.Card) {
//
//}