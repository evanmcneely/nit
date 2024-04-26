package review

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

func TestShouldReviewPullRequest(t *testing.T) {
	t.Run("should ignore pull requests made by bots", func(t *testing.T) {
		event := &github.PullRequestEvent{
			Action: github.String("opened"),
			PullRequest: &github.PullRequest{
				Body:  github.String("body"),
				Title: github.String("title"),
				User: &github.User{
					Login: github.String("something[bot]"), // should ignore
				},
			},
		}

		ok, _ := ShouldReviewPullRequest(event)
		assert.False(t, ok)
	})

	t.Run("should ignore pull requests that are not status opened", func(t *testing.T) {
		createEvent := func(action string) *github.PullRequestEvent {
			return &github.PullRequestEvent{
				Action: github.String(action),
				PullRequest: &github.PullRequest{
					Body:  github.String("body"),
					Title: github.String("title"),
					User: &github.User{
						Login: github.String("user"),
					},
				},
			}
		}

		ignoredActions := []string{
			"assigned",
			"auto_merge_disabled",
			"auto_merge_enabled",
			"closed",
			"converted_to_draft",
			"demilestoned",
			"dequeued",
			"edited",
			"enqueued",
			"labeled",
			"locked",
			"milestoned",
			"ready_for_review",
			"reopened",
			"review_request_removed",
			"review_requested",
			"synchronize",
			"unassigned",
			"unlabeled",
			"unlocked",
		}

		for _, action := range ignoredActions {
			event := createEvent(action)
			ok, _ := ShouldReviewPullRequest(event)
			assert.False(t, ok)
		}
	})

	t.Run("should ignore pull requests marked to be ignored", func(t *testing.T) {
		event := &github.PullRequestEvent{
			Action: github.String("opened"),
			PullRequest: &github.PullRequest{
				Body:  github.String("bla bla ai-review:ignore bla bla"), // should ignore
				Title: github.String("title"),
				User: &github.User{
					Login: github.String("user"),
				},
			},
		}

		ok, _ := ShouldReviewPullRequest(event)
		assert.False(t, ok)
	})
}

func TestReviewPullRequest(t *testing.T) {
	var (
		number            = 123
		title             = "title"
		description       = "bla bla bla"
		mockNotes         = "notes"
		simpleMockDiff    = "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\nline 1\n-remove line 2\n+new line 2\n+add line 3"
		simpleMockPayload = "{\"body\": \"bla bla bla pr body bla bla\", \"event\": \"APPROVE\", \"comments\": [{\"path\": \"file.txt\", \"position\": 1, \"body\": \"Contructive comment...\"}]}"
	)

	// A github event payload that should be reviewed
	event := &github.PullRequestEvent{
		Action: github.String("opened"),
		Repo: &github.Repository{
			Name: github.String("repo"),
			Owner: &github.User{
				Login: github.String("user"),
			},
		},
		PullRequest: &github.PullRequest{
			Body:   github.String(description),
			Title:  github.String(title),
			Number: github.Int(number),
			User: &github.User{
				Login: github.String("user"),
			},
		},
	}

	t.Run("should return a generated review for a PR", func(t *testing.T) {
		setupProviderMock := func(res ...*completionResponse) AIProviderMock {
			calls := 0
			return AIProviderMock{
				CreateCompletetionFunc: func(req *completionRequest) (*completionResponse, error) {
					r := res[calls]
					calls++
					return r, nil
				},
			}
		}

		setupGithubMock := func(assertions ...ghMock.MockBackendOption) *github.Client {
			mockGithub := ghMock.NewMockedHTTPClient(assertions...)
			return github.NewClient(mockGithub)
		}

		type TestReviewContext struct {
			Diff    string
			Payload string
			Want    *github.PullRequestReviewRequest
		}

		tests := []TestReviewContext{
			// simple case: diff contains one hunk in one file and the generated payload is valid
			{
				Diff:    simpleMockDiff,
				Payload: simpleMockPayload,
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(1),
						},
					},
				},
			},
			// diff contains one hunk in the same file and the generated payload is valid
			{
				Diff:    "diff --git a/file.txt b/file.txt\ndeleted file mode 100644\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ /dev/null\n@@ -1,2 +1,3 @@\n-deleted line 1\n-deleted line 2\n-deleted line 3",
				Payload: simpleMockPayload,
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(1),
						},
					},
				},
			},
			// diff contains multiple hunks in the same file and the generated payload is valid
			{
				Diff:    "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\n-deleted line 1\n-deleted line 2\n-deleted line 3\n@@ -1,2 +1,3 @@\n-deleted line 1\n-deleted line 2\n-deleted line 3",
				Payload: simpleMockPayload,
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(1),
						},
					},
				},
			},
			// diff contains a renamed file and the generated payload is valid
			{
				Diff:    "diff --git a/file.txt b/fileNew.txt\nsimilarity index 50%\n rename from file.txt\nrename to fileNew.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/fileNew.txt\n@@ -1,2 +1,3 @@\nline 1\n+add line 2\n-deleted line 3",
				Payload: simpleMockPayload,
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(1),
						},
					},
				},
			},
			// simple diff with multiple valid comments
			{
				Diff:    simpleMockDiff,
				Payload: "{\"body\": \"bla bla bla pr body bla bla\", \"event\": \"APPROVE\", \"comments\": [{\"path\": \"file.txt\", \"position\": 1, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 2, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 3, \"body\": \"Contructive comment...\"}]}",
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(1),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(2),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(3),
						},
					},
				},
			},
			// simple diff with invalid file name in the generated payload
			{
				Payload: "{\"body\": \"bla bla bla pr body bla bla\", \"event\": \"APPROVE\", \"comments\": [{\"path\": \"nope.txt\", \"position\": 1, \"body\": \"Contructive comment...\"}]}",
				Diff:    simpleMockDiff,
				Want: &github.PullRequestReviewRequest{
					Body:     github.String("bla bla bla pr body bla bla"),
					Event:    github.String("APPROVE"),
					Comments: nil, // comment is removed - we actually send and empty array but the value gets niled in tests
				},
			},
			// simple diff with invalid comment position in the generated payload
			{
				Payload: "{\"body\": \"bla bla bla pr body bla bla\", \"event\": \"APPROVE\", \"comments\": [{\"path\": \"file.txt\", \"position\": 5, \"body\": \"Contructive comment...\"}]}",
				Diff:    simpleMockDiff,
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(4),
						},
					},
				},
			},
			// a diff with a deleted file and the generated payload is valid
			{
				Payload: "{\"body\": \"bla bla bla pr body bla bla\", \"event\": \"APPROVE\", \"comments\": [{\"path\": \"file.txt\", \"position\": 4, \"body\": \"Contructive comment...\"}]}",
				Diff:    "diff --git a/file.txt b/file.txt\ndeleted file mode 100644\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ /dev/null\n@@ -1,2 +1,3 @@\n-deleted line 1\n-deleted line 2\n-deleted line 3",
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(3),
						},
					},
				},
			},
			// simple diff with multiple comments in the generated payload, one of which is invalid (invalid comment is in the middle of the comments array)
			{
				Diff:    simpleMockDiff,
				Payload: "{\"body\": \"bla bla bla pr body bla bla\", \"event\": \"APPROVE\", \"comments\": [{\"path\": \"file.txt\", \"position\": 1, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 2, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 5, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 3, \"body\": \"Contructive comment...\"}]}",
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(1),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(2),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(4),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(3),
						},
					},
				},
			},
			// simple diff with multiple comments in the generated payload and multiple invalid comments
			{
				Payload: "{\"body\": \"bla bla bla pr body bla bla\", \"event\": \"APPROVE\", \"comments\": [{\"path\": \"file.txt\", \"position\": 15, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 2, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 6, \"body\": \"Contructive comment...\"},{\"path\": \"file.txt\", \"position\": 5, \"body\": \"Contructive comment...\"}]}",
				Diff:    simpleMockDiff,
				Want: &github.PullRequestReviewRequest{
					Body:  github.String("bla bla bla pr body bla bla"),
					Event: github.String("APPROVE"),
					Comments: []*github.DraftReviewComment{
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(4),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(2),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(4),
						},
						{
							Path:     github.String("file.txt"),
							Body:     github.String("Contructive comment..."),
							Position: github.Int(4),
						},
					},
				},
			},
		}

		for _, c := range tests {
			mockProvider := setupProviderMock(
				// return a mock response from generating review notes
				&completionResponse{
					Completion: mockNotes,
					Tokens:     10,
				},
				// return a mock response from generating the review body
				&completionResponse{
					Completion: c.Payload,
					Tokens:     20,
				},
			)
			mockAI := NewAI(&mockProvider, &mockProvider)

			var reviewPayload *github.PullRequestReviewRequest
			mockGithub := setupGithubMock(
				[]ghMock.MockBackendOption{
					// return the pull request diff
					ghMock.WithRequestMatchHandler(
						ghMock.GetReposPullsByOwnerByRepoByPullNumber,
						http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.Write([]byte(c.Diff))
						}),
					),
					// return a successful response from posting the pull request review
					ghMock.WithRequestMatchHandler(
						ghMock.PostReposPullsReviewsByOwnerByRepoByPullNumber,
						http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							json.NewDecoder(r.Body).Decode(&reviewPayload)
							w.Write([]byte("")) // response doesn't matter
						}),
					),
				}...,
			)

			// should return no errors
			ok := ReviewPullRequest(event, mockAI, mockGithub)
			assert.Nil(t, ok)

			// assert that the payload "sent" to Github was formed properly
			assert.Equal(t, c.Want, reviewPayload)

			// assert that the call to generate review notes is formed correctly
			gotAI1 := mockProvider.calls.CreateCompletetion[0].Req
			wantAI1 := &completionRequest{
				Prompt: fmt.Sprintf(reviewCommentsPrompt, formatPullRequestDetails(number, title, description), mockAI.addPositionNumbersToDiff(c.Diff)),
				Model:  modelGood,
				Format: formatText,
			}
			assert.Equal(t, wantAI1, gotAI1)

			// assert that the call to generate review payload is formed correctly
			gotAI2 := mockProvider.calls.CreateCompletetion[1].Req
			wantAI2 := &completionRequest{
				Prompt: fmt.Sprintf(reviewPostBodyPrompt, formatPullRequestDetails(number, title, description), mockNotes),
				Model:  modelCheap,
				Format: formatJSON,
			}
			assert.Equal(t, wantAI2, gotAI2)
		}
	})

	t.Run("should return AI provider errors when generating review notes", func(t *testing.T) {
		mockGithub := github.NewClient(ghMock.NewMockedHTTPClient())
		mockProvider := AIProviderMock{
			CreateCompletetionFunc: func(req *completionRequest) (*completionResponse, error) {
				return nil, errors.New("something happened")
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		ok := ReviewPullRequest(event, mockAI, mockGithub)
		assert.Error(t, ok)
	})

	t.Run("should return AI provider errors when generating payload", func(t *testing.T) {
		firstCallDone := false
		mockGithub := github.NewClient(ghMock.NewMockedHTTPClient())
		mockProvider := AIProviderMock{
			CreateCompletetionFunc: func(req *completionRequest) (*completionResponse, error) {
				if firstCallDone {
					return nil, errors.New("something happened")
				}
				// return an error on the second call to the provider
				return &completionResponse{
					Completion: mockNotes,
					Tokens:     10,
				}, nil
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		ok := ReviewPullRequest(event, mockAI, mockGithub)
		assert.Error(t, ok)
	})

	t.Run("should return Github errors when fetching the diff fails", func(t *testing.T) {
		mockedHTTPClient := ghMock.NewMockedHTTPClient(
			ghMock.WithRequestMatchHandler(
				// request to get the pull request diff fails
				ghMock.GetReposPullsByOwnerByRepoByPullNumber,
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
			CreateCompletetionFunc: func(req *completionRequest) (*completionResponse, error) {
				return &completionResponse{
					Completion: "bla bla bla",
					Tokens:     10,
				}, nil
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		ok := ReviewPullRequest(event, mockAI, mockGithub)
		assert.Error(t, ok)
	})

	t.Run("should return Github errors when posting the pull request review fails", func(t *testing.T) {
		mockedHTTPClient := ghMock.NewMockedHTTPClient(
			ghMock.WithRequestMatchHandler(
				// return the diff successfully
				ghMock.GetReposPullsByOwnerByRepoByPullNumber,
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Write([]byte(""))
				}),
			),
			ghMock.WithRequestMatchHandler(
				// fail when post the review payload
				ghMock.PostReposPullsReviewsByOwnerByRepoByPullNumber,
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
			CreateCompletetionFunc: func(req *completionRequest) (*completionResponse, error) {
				// always return something successfully
				return &completionResponse{
					Completion: "bla bla bla",
					Tokens:     10,
				}, nil
			},
		}
		mockAI := NewAI(&mockProvider, &mockProvider)

		ok := ReviewPullRequest(event, mockAI, mockGithub)
		assert.Error(t, ok)
	})
}

func TestAddPositionNumbersToDiff(t *testing.T) {
	t.Run("should add position numbers to diff with one file and one hunk", func(t *testing.T) {
		ai := AI{}

		diff := "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\nline 1\n-remove line 2\n+new line 2\n+add line 3"
		want := "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\n1 line 1\n2 -remove line 2\n3 +new line 2\n4 +add line 3"
		got := ai.addPositionNumbersToDiff(diff)

		assert.Equal(t, want, got)
	})

	t.Run("should add position numbers to diff with one file and multiple hunks", func(t *testing.T) {
		ai := AI{}

		diff := "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\nline 1\n-remove line 2\n+new line 2\n+add line 3\n@@ -1,2 +1,3 @@\nline 6\n-remove line 7\n+new line 7\n+add line 9"
		want := "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\n1 line 1\n2 -remove line 2\n3 +new line 2\n4 +add line 3\n5 @@ -1,2 +1,3 @@\n6 line 6\n7 -remove line 7\n8 +new line 7\n9 +add line 9"
		got := ai.addPositionNumbersToDiff(diff)

		assert.Equal(t, want, got)
	})

	t.Run("should add position numbers to diff with multiple files", func(t *testing.T) {
		ai := AI{}

		diff := "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\nline 1\n-remove line 2\n+new line 2\n+add line 3\ndiff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file2.txt\n+++ b/file2.txt\n@@ -1,2 +1,3 @@\nline 1\n-remove line 2\n+new line 2\n+add line 3"
		want := "diff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,3 @@\n1 line 1\n2 -remove line 2\n3 +new line 2\n4 +add line 3\ndiff --git a/file.txt b/file.txt\nindex 123456789..123456789 100644\n--- a/file2.txt\n+++ b/file2.txt\n@@ -1,2 +1,3 @@\n1 line 1\n2 -remove line 2\n3 +new line 2\n4 +add line 3"
		got := ai.addPositionNumbersToDiff(diff)

		assert.Equal(t, want, got)
	})
}
