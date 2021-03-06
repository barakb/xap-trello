package xap_trello

import (
	"net/http"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route


var burndown *Burndown
var routes  Routes

func InitRouters(){
	burndown = NewBurnDown()
	routes = Routes{
		Route{
			"GET_TIMELINE",
			"GET",
			"/api/timeline",
			CreateTimelineHandler(burndown),
		},
		Route{
			"VIEW",
			"GET",
			"/",
			CreateViewHandler(),
		},
		Route{
			"NEXT_SPRINT",
			"GET",
			"/api/sprint/next",
			CreateGuessSprintParamsHandler(),
		},
		Route{
			"SAVE",
			"POST",
			"/api/sprint/next",
			CreateNextSprintHandler(burndown),
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

}





