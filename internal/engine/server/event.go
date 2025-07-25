package server

import (
	"encoding/json"
	"io"
	"net/http"
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var eventPayload WebhookPayload
	err = json.Unmarshal(body, &eventPayload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
