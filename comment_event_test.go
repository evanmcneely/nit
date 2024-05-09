package nit

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-github/v59/github"
	ghMock "github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
)

func TestShouldRespondToComment(t *testing.T) {
	const appName = "app"

	setupGithubMocks := func(assertions ...ghMock.MockBackendOption) *github.Client {
		mockGithub := ghMock.NewMockedHTTPClient(assertions...)
		return github.NewClient(mockGithub)
	}

	t.Run("should ignore comments not made in response to our own comments", func(t *testing.T) {
		mockGh := setupGithubMocks(
			ghMock.WithRequestMatch(
				ghMock.GetReposPullsCommentsByOwnerByRepoByCommentId,
				github.PullRequestComment{
					User: &github.User{
						Login: github.String("something"), // should ignore
					},
				},
			),
		)

		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String("something"),
				},
				InReplyTo: github.Int64(123),
			},
		}

		ok, _ := ShouldRespondToComment(event, mockGh, &Config{AppName: appName})
		assert.False(t, ok)
	})

	t.Run("should ignore comments that are not of action created", func(t *testing.T) {
		mockGh := setupGithubMocks()

		ignoredActions := []string{
			"deleted",
			"edited",
		}

		getIgonredEvent := func(action string) *github.PullRequestReviewCommentEvent {
			return &github.PullRequestReviewCommentEvent{
				Action: github.String(action),
				Repo: &github.Repository{
					Name: github.String("repo"),
					Owner: &github.User{
						Login: github.String("user"),
					},
				},
				Comment: &github.PullRequestComment{
					User: &github.User{
						Login: github.String("something"),
					},
					InReplyTo: github.Int64(123),
				},
			}
		}

		for _, action := range ignoredActions {
			event := getIgonredEvent(action)
			ok, _ := ShouldRespondToComment(event, mockGh, &Config{AppName: appName})
			assert.False(t, ok)
		}
	})

	t.Run("should ignore comments not in reply to anything", func(t *testing.T) {
		mockGh := setupGithubMocks()

		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String("something"),
				},
				// InReplyTo is absent
			},
		}

		ok, _ := ShouldRespondToComment(event, mockGh, &Config{AppName: appName})
		assert.False(t, ok)
	})

	t.Run("should ignore pull requests made by bots", func(t *testing.T) {
		mockGh := setupGithubMocks()

		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String("something[bot]"), // should ignore
				},
				InReplyTo: github.Int64(123),
			},
		}

		ok, _ := ShouldRespondToComment(event, mockGh, &Config{AppName: appName})
		assert.False(t, ok)
	})
}

func TestRespondToComment(t *testing.T) {
	var (
		appName         = "my-app"
		pr              = 123
		user            = "something"
		commentId int64 = 456
		comment         = "bla bla bla"
		hunk            = "hunky"
		reply           = "wa wa wa"
		comments        = []*github.PullRequestComment{
			{
				User: &github.User{
					Login: github.String("user"),
				},
				DiffHunk: github.String(hunk),
				Body:     github.String(comment),
			},
		}
	)

	t.Run("should reply to a comment in thread", func(t *testing.T) {
		mockProvider := AIProviderMock{
			CreateCompletetionFunc: func(req *completionRequest) (*CompletionResponse, error) {
				// respond with the reply
				return &CompletionResponse{Completion: reply, Tokens: 10}, nil
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		var commentPayload *github.PullRequestComment
		mockedHTTPClient := ghMock.NewMockedHTTPClient(
			ghMock.WithRequestMatch(
				// request to get all the comments in the thread returns our comment
				ghMock.GetReposPullsCommentsByOwnerByRepoByPullNumber,
				comments,
			),
			ghMock.WithRequestMatchHandler(
				// request to post the comment reply successful
				ghMock.PostReposPullsCommentsByOwnerByRepoByPullNumber,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					json.NewDecoder(r.Body).Decode(&commentPayload)
					w.Write([]byte("")) // response doesn't matter
				}),
			),
		)
		mockGithub := github.NewClient(mockedHTTPClient)

		// for a pull request comment event that needs a response
		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			PullRequest: &github.PullRequest{
				Number: github.Int(pr),
			},
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String(user),
				},
				InReplyTo: github.Int64(commentId),
				DiffHunk:  github.String(hunk),
				Body:      github.String(comment),
			},
		}

		// should return no errors
		_, ok := RespondToComment(event, &Config{AppName: appName}, mockAI, mockGithub)
		assert.Nil(t, ok)

		// assert that the payload "sent" to Github was formed properly
		wantGithub := &github.PullRequestComment{
			Body: github.String(reply),
		}
		assert.Equal(t, wantGithub, commentPayload)

		// assert that the call to generate reply is formed correctly
		gotAI := mockProvider.calls.CreateCompletetion[0].Req
		wantAI := &completionRequest{
			Prompt: fmt.Sprintf(commentReplyPrompt, comment, hunk, appName, formatPullRequestComments(comments)),
			Model:  modelGood,
			Format: formatText,
		}
		assert.Equal(t, wantAI, gotAI)
	})

	t.Run("should not post a reply if the model doesn't want to", func(t *testing.T) {
		mockProvider := AIProviderMock{
			CreateCompletetionFunc: func(req *completionRequest) (*CompletionResponse, error) {
				// respond with the reply
				return &CompletionResponse{Completion: noreply, Tokens: 10}, nil
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		mockedHTTPClient := ghMock.NewMockedHTTPClient(
			ghMock.WithRequestMatch(
				// request to get all the comments in the thread returns our comments
				ghMock.GetReposPullsCommentsByOwnerByRepoByPullNumber,
				comments,
			),
		)
		mockGithub := github.NewClient(mockedHTTPClient)

		// for a pull request comment event that needs a response
		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			PullRequest: &github.PullRequest{
				Number: github.Int(pr),
			},
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String(user),
				},
				InReplyTo: github.Int64(commentId),
				DiffHunk:  github.String(hunk),
				Body:      github.String(comment),
			},
		}

		// should return no errors
		_, ok := RespondToComment(event, &Config{AppName: appName}, mockAI, mockGithub)
		assert.Nil(t, ok)
	})

	t.Run("should return AI provider errors when generating reply fails", func(t *testing.T) {
		mockGithub := github.NewClient(ghMock.NewMockedHTTPClient())
		mockProvider := AIProviderMock{
			CreateCompletetionFunc: func(req *completionRequest) (*CompletionResponse, error) {
				return nil, errors.New("something happened")
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		// for an pull request comment event that needs a response
		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			PullRequest: &github.PullRequest{
				Number: github.Int(pr),
			},
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String(user),
				},
				InReplyTo: github.Int64(commentId),
				DiffHunk:  github.String(hunk),
				Body:      github.String(comment),
			},
		}

		_, ok := RespondToComment(event, &Config{AppName: appName}, mockAI, mockGithub)
		assert.Error(t, ok)
	})

	t.Run("should return Github errors when fetching the comment thread fails", func(t *testing.T) {
		mockedHTTPClient := ghMock.NewMockedHTTPClient(
			ghMock.WithRequestMatchHandler(
				// request to get comment thread fails
				ghMock.GetReposPullsCommentsByOwnerByRepoByPullNumber,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ghMock.WriteError(
						w,
						http.StatusInternalServerError,
						"github went belly up or something",
					)
				}),
			),
		)
		mockGithub := github.NewClient(mockedHTTPClient)

		// there should be no calls to an AI model
		mockProvider := AIProviderMock{}
		mockAI := NewAI(&mockProvider, &mockProvider)

		// for an pull request comment event that needs a response
		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			PullRequest: &github.PullRequest{
				Number: github.Int(pr),
			},
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String(user),
				},
				InReplyTo: github.Int64(commentId),
				DiffHunk:  github.String(hunk),
				Body:      github.String(comment),
			},
		}

		_, ok := RespondToComment(event, &Config{AppName: appName}, mockAI, mockGithub)
		assert.Error(t, ok)
	})

	t.Run("should return Github errors when posting the comment fails", func(t *testing.T) {
		mockedHTTPClient := ghMock.NewMockedHTTPClient(
			ghMock.WithRequestMatch(
				// fetching the comment thread is successful
				ghMock.GetReposPullsCommentsByOwnerByRepoByPullNumber,
				comments,
			),
			ghMock.WithRequestMatchHandler(
				// posting the comment fails
				ghMock.PostReposPullsCommentsByOwnerByRepoByPullNumber,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ghMock.WriteError(
						w,
						http.StatusInternalServerError,
						"github went belly up or something",
					)
				}),
			),
		)

		mockGithub := github.NewClient(mockedHTTPClient)
		mockProvider := AIProviderMock{
			CreateCompletetionFunc: func(req *completionRequest) (*CompletionResponse, error) {
				// always return something successfully
				return &CompletionResponse{
					Completion: reply,
					Tokens:     10,
				}, nil
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		// for an pull request comment event that needs a response
		event := &github.PullRequestReviewCommentEvent{
			Action: github.String("created"),
			PullRequest: &github.PullRequest{
				Number: github.Int(pr),
			},
			Repo: &github.Repository{
				Name: github.String("repo"),
				Owner: &github.User{
					Login: github.String("user"),
				},
			},
			Comment: &github.PullRequestComment{
				User: &github.User{
					Login: github.String(user),
				},
				InReplyTo: github.Int64(commentId),
				DiffHunk:  github.String(hunk),
				Body:      github.String(comment),
			},
		}

		_, ok := RespondToComment(event, &Config{AppName: appName}, mockAI, mockGithub)
		assert.Error(t, ok)
	})
}
