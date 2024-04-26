package review

import (
	"context"
	"strings"

	"github.com/google/go-github/v59/github"
)

const maxReviewAttempts = 3

func ShouldReviewPullRequest(e *github.PullRequestEvent) (bool, string) {
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
	default:
		return true, ""
	}
}

func ReviewPullRequest(event *github.PullRequestEvent, ai *AI, gh *github.Client) error {
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
		return err
	}

	body, err := ai.GeneratePullRequestReview(number, title, description, diff)
	if err != nil {
		return err
	}

	// TODO: handle rate limit errors
	_, _, err = gh.PullRequests.CreateReview(
		context.Background(),
		owner,
		repository,
		number,
		body,
	)
	if err != nil {
		return err
	}

	return nil
}
