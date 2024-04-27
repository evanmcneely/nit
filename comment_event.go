package nit

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v59/github"
)

const noreply = "noreply"

func ShouldRespondToComment(e *github.PullRequestReviewCommentEvent, client *github.Client, config *Config) (bool, string) {
	var (
		action     = e.GetAction()
		author     = e.GetComment().GetUser().GetLogin()
		owner      = e.GetRepo().GetOwner().GetLogin()
		repository = e.GetRepo().GetName()
		inReplyTo  = e.GetComment().GetInReplyTo()
	)

	switch {
	// comments on our PR reviews will always have a reply-to
	case inReplyTo == 0:
		return false, "comment is not in reply"
	// ignore comments by bots (including our own app)
	// the [bot] postfix is added by github to app accounts
	case strings.Contains(author, "[bot]"):
		return false, "comment made by our own app"
	// we will only handle comments that are just created (not edited, deleted, etc)
	case action != "created":
		return false, "comment was not \"created\""
	}

	// TODO: handle rate limiting
	origComment, _, err := client.PullRequests.GetComment(context.Background(), owner, repository, inReplyTo)
	if err != nil {
		return false, fmt.Sprintf("could not retrieve original comment: %v", err)
	}

	origPoster := origComment.User.GetLogin()
	if !strings.Contains(origPoster, config.AppName) {
		return false, "comment is not in response to our app"
	}

	return true, ""
}

func RespondToComment(event *github.PullRequestReviewCommentEvent, config *Config, ai *AI, gh *github.Client) error {
	var (
		owner      = event.GetRepo().GetOwner().GetLogin()
		repository = event.GetRepo().GetName()
		pr         = event.GetPullRequest().GetNumber()
		body       = event.GetComment().GetBody()
		hunk       = event.GetComment().GetDiffHunk()
		inReplyTo  = event.GetComment().GetInReplyTo()
	)

	comments, err := getCommentsOnHunk(owner, repository, hunk, pr, gh)
	if err != nil {
		return err
	}

	reply, err := ai.GenerateCommentReply(
		body,
		hunk,
		comments,
		config.AppName,
	)
	if err != nil || reply == nil {
		return err
	}
	if *reply == noreply {
		return nil
	}

	_, _, err = gh.PullRequests.CreateCommentInReplyTo(
		context.Background(),
		owner,
		repository,
		pr,
		*reply,
		inReplyTo,
	)
	if err != nil {
		return err
	}
	return nil
}

func getCommentsOnHunk(owner, repo, hunk string, number int, gh *github.Client) ([]*github.PullRequestComment, error) {
	// todo: paginate this to get all comments
	allComments, _, err := gh.PullRequests.ListComments(
		context.Background(),
		owner,
		repo,
		number,
		&github.PullRequestListCommentsOptions{},
	)
	if err != nil {
		return nil, err
	}

	filterComments := []*github.PullRequestComment{}
	for _, comment := range allComments {
		// Use the current diff hunk that the comment pertains to to filter out comments
		// not apart of the current thread. This is inefficient because hunks can be very
		// large strings. Works for now.
		if comment.DiffHunk != nil && *comment.DiffHunk != hunk {
			continue
		}
		filterComments = append(filterComments, comment)
	}

	return filterComments, nil
}
