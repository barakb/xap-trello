package xap_trello

import (
	"net/http"
	"html/template"
	"fmt"
	"time"
	"encoding/json"
	"log"
)

func CreateTimelineHandler(burndown *Burndown) http.HandlerFunc {
	const ETAG_SERVER_HEADER = "ETag"
	const ETAG_CLIENT_HEADER = "If-None-Match"
	return func(w http.ResponseWriter, r *http.Request) {
		sprintStatus := burndown.GetSprintStatus()
		etag := fmt.Sprintf("ver:%d:%s", sprintStatus.Version, toDayStr(time.Now()))
		if match := r.Header.Get(ETAG_CLIENT_HEADER); match != "" {
			if match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		//w.Header().Set("Cache-Control", "max-age=20")
		w.Header().Set(ETAG_SERVER_HEADER, etag)
		//compressed := compressTimeline(timeline)
		//sprint := createSprint(listWatcher.sprint, compressed, totalPoints)

		if err := json.NewEncoder(w).Encode(sprintStatus); err != nil {
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


func  indexOf(strings []string, value string) int {
	for p, v := range strings {
		if (v == value) {
			return p
		}
	}
	return -1
}