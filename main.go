package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/evanmcneely/nit/internal/config"
	"github.com/evanmcneely/nit/pkg/nit"
	"github.com/google/go-github/v59/github"
)

// Untested...
func main() {
	// Load the config
	config, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize AI providers
	openai := nit.NewOpenAI(config.AI.OpenaiKey)
	ai := nit.NewAI(openai, openai)

	// Initialize Github client
	gh := github.NewClient(nil)

	// Define the handler function.
	http.HandleFunc("webhook/github", HandleGithubEvents(&config, ai, gh))

	// Start the server
	log.Printf("server starting on port %v...", config.App.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%v", config.App.Port), nil)
	if err != nil {
		log.Printf("error starting server: %v", err)
	}
}

// Handle Github webhook events for Pull Requests and Pull Request Comments
func HandleGithubEvents(c *config.Config, ai *nit.AI, gh *github.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		payload, err := github.ValidatePayload(r, []byte(c.Github.WebhookSecret))
		if err != nil {
			log.Printf("could not validate payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			log.Printf("could not parse webhook: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Acknowledge receipt of the payload
		w.WriteHeader(http.StatusNoContent)

		switch event := event.(type) {
		case *github.PullRequestEvent:
			if ok, reason := nit.ShouldReviewPullRequest(event); !ok {
				log.Printf("not reviewing pull request because: %v", reason)
			}
			err = nit.ReviewPullRequest(event, ai, gh)
			if err != nil {
				log.Printf("error reviewing pull request: %v", err)
			}
		case *github.PullRequestReviewCommentEvent:
			if ok, reason := nit.ShouldRespondToComment(event, gh, c.App.Name); !ok {
				log.Printf("not replying to comment because: %v", reason)
			}
			err = nit.RespondToComment(event, c.App.Name, ai, gh)
			if err != nil {
				log.Printf("error replying to comment: %v", err)
			}
		default:
			log.Printf("ignoring event %s", r.Header.Get("X-GitHub-Event"))
		}
	}
}
