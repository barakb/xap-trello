package xap_trello

import (
	"net/http"
	"github.com/barakb/go-trello"
	"time"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var listWatcher = NewWatcher(func(l trello.List) bool {
	return l.Name == "Done in m6!"
}, 10 * time.Second)

var routes = Routes{
	Route{
		"GET_TIMELINE",
		"GET",
		"/api/timeline",
		CreateTimelineHandler(listWatcher),
	},
	Route{
		"VIEW",
		"GET",
		"/",
		CreateViewHandler(),
	},
	//Route{
	//	"CFG.ADD.MACHINES",
	//	"PUT",
	//	"/api/cfg/machines/{name}",
	//	CFGAddMachine,
	//},
	//Route{
	//	"CFG.GET",
	//	"GET",
	//	"/api/cfg",
	//	CFGGet,
	//},
}

