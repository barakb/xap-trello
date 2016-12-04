package xap_trello

import (
	"net/http"
	"html/template"
	"fmt"
	"time"
	"encoding/json"
	"log"
	"regexp"
	"strconv"
)

const date_tmpl = "2006-01-02"

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

func CreateNextSprintHandler(burndown *Burndown) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("reading sprint data")
		decoder := json.NewDecoder(r.Body)
		sprintParams := SprintParams{}
		err := decoder.Decode(&sprintParams)
		defer r.Body.Close()
		if err != nil {
			log.Printf("error %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("sprint params are: %q\n", sprintParams)
		var name string = sprintParams.Name
		if name != "" {
			log.Printf("missing sprint name %+v\n", sprintParams)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		start, err := time.Parse(date_tmpl, sprintParams.Start)
		if err != "" {
			log.Printf("error %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		end, err := time.Parse(date_tmpl, sprintParams.Start)
		if err != "" {
			log.Printf("error %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		<- burndown.startNewSprint(name, start, end)
		w.WriteHeader(http.StatusOK)
	}
}

func CreateViewHandler() http.HandlerFunc {
	t := template.Must(template.ParseFiles("index.html"))
	return func(w http.ResponseWriter, r *http.Request) {
		t.Execute(w, r)
	}
}

type SprintParams struct {
	Name  string `json:"name"`
	Start string `json:"start"`
	End   string `json:"end"`
}

func CreateGuessSprintParamsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start, end, name, err := getNextSprintDefaults()
		if err != nil {
			log.Printf("error %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sp := &SprintParams{Name:name, Start:start.Format(date_tmpl), End:end.Format(date_tmpl)}
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		if err := json.NewEncoder(w).Encode(sp); err != nil {
			log.Printf("error %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	}
}

func indexOf(strings []string, value string) int {
	for p, v := range strings {
		if (v == value) {
			return p
		}
	}
	return -1
}

func getNextSprintDefaults() (start, end time.Time, name string, err error) {
	xapOpenJira, err := CreateXAPJiraOpen()
	if err != nil {
		return
	}
	sprint, _, err := xapOpenJira.Client.Board.GetLastSprint(fmt.Sprintf("%d", xapOpenJira.MainScrumBoardId))
	start = sprint.StartDate.AddDate(0, 0, 7)
	end = sprint.EndDate.AddDate(0, 0, 7)
	name, err = suggestNextSprintName(sprint.Name)
	fmt.Printf("Sprint is %+v\n", sprint)
	return
}

func suggestNextSprintName(prev string) (string, error) {
	milestonePattern := regexp.MustCompile(`(?i)(.*)-M([0-9]+)`)
	match := milestonePattern.FindStringSubmatch(prev)
	if match != nil {
		i, err := strconv.Atoi(match[2])
		if err != nil {
			return "", fmt.Errorf("Failed to convert %q to int, name is %q\n", match[2], prev)
		}
		return fmt.Sprintf("%s-M%d", match[1], i + 1), nil
	}
	rcPattern := regexp.MustCompile(`(?i)(.*)-rc([0-9]+)`)
	match = rcPattern.FindStringSubmatch(prev)
	if match != nil {
		i, err := strconv.Atoi(match[2])
		if err != nil {
			return "", fmt.Errorf("Failed to convert %q to int, name is %q\n", match[2], prev)
		}
		return fmt.Sprintf("%s-rc%d", match[1], i + 1), nil
	}
	return prev, nil
}