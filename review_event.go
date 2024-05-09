package nit

import (
	"context"
	"strings"

	"github.com/google/go-github/v59/github"
)

const maxReviewAttempts = 3

type ReviewResponse struct {
	Tokens int
	Id     int64
}

func ShouldReviewPullRequest(e *github.PullRequestEvent, c *Config) (bool, string) {
	var (
		author      = e.GetPullRequest().GetUser().GetLogin()
		action      = e.GetAction()
		description = e.GetPullRequest().GetBody()
	)

	switch {
	// ignore pull requests oppened by bots
	// the [bot] postfix is added by github to app accounts
	case strings.Contains(author, "[bot]"):
		return false, "pull request made by a bot"
	// we will only handle pull requests that are just opened
	case action != "opened":
		return false, "pull request was not \"opened\""
	// the ai-review:ignore string can be added to the PR description by a dev to
	// prevent it from being revieed by this app
	case strings.Contains(description, "ai-review:ignore"):
		return false, "pull request marked as ignore"
	// the ai-review:please string must be added to PR descriptions by a dev when
	// OptIn is set to true
	case c.OptIn && !strings.Contains(description, "ai-review:please"):
		return false, "review not requested when opt-in is enabled"
	default:
		return true, ""
	}
}

func ReviewPullRequest(event *github.PullRequestEvent, ai *AI, gh *github.Client) (*ReviewResponse, error) {
	var (
		owner       = event.GetRepo().GetOwner().GetLogin()
		repository  = event.GetRepo().GetName()
		number      = event.GetPullRequest().GetNumber()
		title       = event.GetPullRequest().GetTitle()
		description = event.GetPullRequest().GetBody()
	)

	// TODO: handle rate limit errors
	diff, _, err := gh.PullRequests.GetRaw(
		context.Background(),
		owner,
		repository,
		number,
		github.RawOptions{Type: github.Diff},
	)
	if err != nil {
		return nil, err
	}

	body, tokens, err := ai.GeneratePullRequestReview(number, title, description, diff)
	if err != nil {
		return nil, err
	}

	// TODO: handle rate limit errors
	review, _, err := gh.PullRequests.CreateReview(
		context.Background(),
		owner,
		repository,
		number,
		body,
	)
	if err != nil {
		return nil, err
	}

	return &ReviewResponse{
		Tokens: tokens,
		Id:     review.GetID(),
	}, nil
}
