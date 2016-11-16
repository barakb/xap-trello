package xap_trello

import (
	"github.com/barakb/go-trello"
	"os"
)

type Trello struct {
	Client *trello.Client
}


func CreateXAPTrello() (*Trello, error){
	appToken, appKey := os.Getenv("trelloAppToken"), os.Getenv("trelloAppKey")
	trelloClient, err := trello.NewAuthClient(appKey, &appToken)
	if err != nil {
		return nil, err
	}
	return &Trello{Client: trelloClient}, nil
}

func (c *Trello) Board(name string) (trello.Board ,error) {
	member, err := c.Client.Member("me")
	if err != nil{
		return trello.Board{}, err
	}
	return member.Board(name)
}



