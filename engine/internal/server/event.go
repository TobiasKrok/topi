package server

import (
	"context"
	"encoding/json"
	"github.com/rabbitmq/amqp091-go"
	"github.com/tobiaskrok/topi/shared/objects"
	"log"
	"net/http"
	"strings"
	"time"
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

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	var eventPayload WebhookPayload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&eventPayload); err != nil {
		http.Error(w, "malformed JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	//log.Printf("event payload: %+v", eventPayload)

	tree, err := e.git.GetTree("tobi", "topi-test", "main", true)
	if err != nil {
		log.Printf("error getting tree: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tree.TotalCount == 0 {
		log.Printf(".topi folder not found in repo '%s' ref=%s", eventPayload.Repo.FullName, eventPayload.Ref)
		return
	}

	log.Printf("Tree entries count: %d", len(tree.Entries))
	for _, entry := range tree.Entries {
		log.Printf("Entry path: %s, type: %s", entry.Path, entry.Type)
	}

	hasWorkflows := false
	for _, entry := range tree.Entries {
		if strings.HasPrefix(entry.Path, ".topi/") {
			if strings.HasSuffix(entry.Path, ".yaml") || strings.HasSuffix(entry.Path, ".yml") {
				hasWorkflows = true
				break
			}
		}
	}
	if !hasWorkflows {
		log.Printf("No workflows found, returning 204")
		w.WriteHeader(http.StatusNoContent)
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
	pubCtx, pubCancel := context.WithTimeout(ctx, 10*time.Second)
	defer pubCancel()

	if err := e.rabbitmq.Channel.PublishWithContext(
		pubCtx,
		"topi",
		"engine.trigger",
		true,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        out,
		}); err != nil {

		log.Printf("error publishing trigger: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	e.db.ExecContext(ctx, `INSERT INTO triggers (trigger) VALUES ($1)`, trigger)
	log.Println("trigger published")
	// TODO handle failed publish, this shouldn't really return an error, a new status in the DB should be added saying that this commit is not queued yet
	w.WriteHeader(http.StatusAccepted)

}
