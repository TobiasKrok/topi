package server

import (
	"encoding/json"
	"github.com/rabbitmq/amqp091-go"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"topi/internal/shared/git"
	"topi/internal/shared/objects"
)

type WebhookPayload struct {
	Ref     string     `json:"ref"`
	After   string     `json:"after"`
	Commits []Commit   `json:"commits"`
	Repo    Repository `json:"repository"`
}

type Commit struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	URL     string `json:"url"`
	Author  struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	Committer struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"committer"`
}

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HtmlURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
}

// Events are Webhooks events from Gitea (or any git instance  but rn only gitea)

func (e *EngineServer) eventHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log.Println("event received")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading request body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var eventPayload WebhookPayload
	err = json.Unmarshal(body, &eventPayload)
	if err != nil {
		log.Printf("error parsing request body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("event payload: %+v", eventPayload)

	tree, err := e.git.GetTree("tobi", "topi-test", "main", false) // check if we have a .topi folder for workflows, we dont need recursive
	if err != nil {
		log.Printf("error getting tree: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tree.TotalCount == 0 {
		log.Printf(".topi folder not found in repo '%s' ref=%s", eventPayload.Repo.FullName, eventPayload.Ref)
		return
	}

	var topiFolder git.TreeEntry
	hasWorkflows := false
	// if .topi folder exists and there are any workflow files in there
	for _, entry := range tree.Entries {
		if entry.Path == ".topi" {
			topiFolder = entry
			continue
		}
		if strings.HasPrefix(entry.Path, ".topi/") && (strings.HasSuffix(entry.Path, ".yaml") || strings.HasSuffix(entry.Path, ".yml")) {
			hasWorkflows = true
			break
		}
	}

	if topiFolder.Path == "" {
		log.Printf(".topi folder not found in repo '%s' ref=%s", eventPayload.Repo.FullName, eventPayload.Ref)
		return
	}

	if !hasWorkflows {
		log.Printf("No workflow files found in .topi folder for repo '%s' ref=%s", eventPayload.Repo.FullName, eventPayload.Ref)
		return
	}
	trigger := objects.BuildTrigger{
		Repository: eventPayload.Repo.HtmlURL,
		Ref:        eventPayload.Ref,
		Commit: objects.BuildCommit{ // TODO multiple commits?
			Sha:       eventPayload.Commits[0].ID,
			Message:   eventPayload.Commits[0].Message,
			Committer: eventPayload.Commits[0].Committer.Username,
		},
		Timestamp: time.Now().Unix(),
	}
	out, _ := json.Marshal(trigger)
	err = e.rabbitmq.Channel.PublishWithContext(r.Context(), "topi", "engine.trigger", false, false, amqp091.Publishing{ContentType: "application/json", Body: out})
	// TODO handle failed publish, this shouldn't really return an error, a new status in the DB should be added saying that this commit is not queued yet

	if err != nil {
		log.Printf("error publishing trigger: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
