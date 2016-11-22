package xap_trello

import (
	"net/http"
	"encoding/json"
	"log"
	"html/template"
	"fmt"
)

func CreateTimelineHandler(listWatcher *ListWatcher) http.HandlerFunc {
	const ETAG_SERVER_HEADER = "ETag"
	const ETAG_CLIENT_HEADER = "If-None-Match"
	return func(w http.ResponseWriter, r *http.Request) {
		timeline := listWatcher.Timeline()
		etag := fmt.Sprintf("%d", len(timeline))
		if match := r.Header.Get(ETAG_CLIENT_HEADER); match != "" {
			if match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Header().Set("Cache-Control", "max-age=20")
		w.Header().Set(ETAG_SERVER_HEADER, etag)

		timeline = compressTimeline(timeline)

		if err := json.NewEncoder(w).Encode(timeline); err != nil {
			log.Printf("error %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func CreateViewHandler() http.HandlerFunc {
	t := template.Must(template.ParseFiles("index.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		t.Execute(w, r)
	}
}


func compressTimeline(events []TimelineEvent) []TimelineEvent {
	m := map[string]TimelineEvent{}
	for _, event := range events{
		m[event.Day] = event
	}
	var res []TimelineEvent = nil
	for _, event := range m{
		res = append(res, event)
	}
	return res
}