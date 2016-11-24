package xap_trello

import (
	"net/http"
	"encoding/json"
	"log"
	"html/template"
	"fmt"
	"github.com/barakb/go-jira"
	"time"
)

func CreateTimelineHandler(listWatcher *ListWatcher) http.HandlerFunc {
	const ETAG_SERVER_HEADER = "ETag"
	const ETAG_CLIENT_HEADER = "If-None-Match"
	return func(w http.ResponseWriter, r *http.Request) {
		timeline, totalPoints := listWatcher.Timeline()
		etag := fmt.Sprintf("%d:%d:%s", len(timeline), totalPoints, toDayStr(time.Now()))
		if match := r.Header.Get(ETAG_CLIENT_HEADER); match != "" {
			if match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		//w.Header().Set("Cache-Control", "max-age=20")
		w.Header().Set(ETAG_SERVER_HEADER, etag)
		compressed := compressTimeline(timeline)
		sprint := createSprint(listWatcher.sprint, compressed, totalPoints)

		if err := json.NewEncoder(w).Encode(*sprint); err != nil {
			log.Printf("error %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

type Day struct {
	Name       string `json:"name"`
	Actual     interface{} `json:"actual"`
	Expected   float64 `json:"expected"`
	WorkingDay bool `json:"working_day"`
}

type Sprint struct {
	Name   string `json:"name"`
	Points int `json:"points"`
	Days   []Day `json:"days"`
	Today int `json:"today"`
}

func createSprint(sprint jira.Sprint, timeline map[string]TimelineEvent, totalPoints int) (s *Sprint) {
	order := []string{}
	m := map[string]bool{}
	date := sprint.StartDate
	for date.Before(*sprint.EndDate) {
		dayStr := toDayStr(*date)
		order = append(order, dayStr)
		m[dayStr] = true
		nextDay := date.Add(24 * time.Hour)
		date = &nextDay
	}
	dayStr := toDayStr(*sprint.EndDate)
	order = append(order, dayStr)
	m[dayStr] = true

	s = &Sprint{Name:sprint.Name, Today:indexOf(order, toDayStr(time.Now()))}
	s.Points = totalPoints

	accPoints := s.Points
	remains := float64(s.Points)
	perDay := remains / float64(len(order))
	for index , day := range order {
		expected := remains - (float64(index + 1) * perDay)
		if e, ok := timeline[day]; ok {
			accPoints -= e.Points
			s.Days = append(s.Days, Day{Name:day, Actual:accPoints, WorkingDay:true, Expected: expected})
		} else if index <= s.Today{
			s.Days = append(s.Days, Day{Name:day, Actual:accPoints, WorkingDay:true, Expected: expected})
		}else{
			s.Days = append(s.Days, Day{Name:day, Actual:nil, WorkingDay:true, Expected: expected})
		}
	}
	s.Days = append([]Day{{"Palnning", s.Points, float64(s.Points), false}},s.Days...)
	return s

}

func CreateViewHandler() http.HandlerFunc {
	t := template.Must(template.ParseFiles("index.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		t.Execute(w, r)
	}
}

func compressTimeline(events []TimelineEvent) map[string]TimelineEvent {
	m := map[string]TimelineEvent{}
	for _, event := range events {
		m[event.Day] = event
	}
	return m
}

func  indexOf(strings []string, value string) int {
	for p, v := range strings {
		if (v == value) {
			return p
		}
	}
	return -1
}