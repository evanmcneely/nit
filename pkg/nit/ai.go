package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v59/github"
)

const (
	modelCheap = "fast"
	modelGood  = "slow"
	formatJSON = "json"
	formatText = "text"
)

//go:generate moq -out mock_AIProvider_test.go . AIProvider

// The interface that an AI provider (such as OpenAI or Anthropic) must implement to be used in this package.
// Fulfilling this interface makes it easier to experiment with different AI providers.
type AIProvider interface {
	CreateCompletetion(req *completionRequest) (*completionResponse, error)
}

type completionRequest struct {
	Model  string
	Prompt string
	Format string
}

type completionResponse struct {
	Completion string
	Tokens     int
}

type AI struct {
	Cheap AIProvider
	Good  AIProvider
}

type completion struct {
	ai       *AI
	model    string
	provider AIProvider
	format   string
}

func (c *completion) Cheap() *completion {
	c.provider = c.ai.Cheap
	c.model = modelCheap
	return c
}

func (c *completion) Good() *completion {
	c.provider = c.ai.Good
	c.model = modelGood
	return c
}

func (c *completion) ReturnJSON() *completion {
	c.format = formatJSON
	return c
}

func (c *completion) ReturnText() *completion {
	c.format = formatText
	return c
}

func (c *completion) Create(prompt string) (*completionResponse, error) {
	if prompt == "" {
		return &completionResponse{}, errors.New("the prompt is empty. aborting completion")
	}

	req := completionRequest{
		Model:  c.model,
		Prompt: prompt,
		Format: c.format,
	}
	return c.provider.CreateCompletetion(&req)
}

func NewAI(good AIProvider, cheap AIProvider) *AI {
	return &AI{
		Good:  good,
		Cheap: cheap,
	}
}

func (ai *AI) NewCompletion() *completion {
	return &completion{
		ai:       ai,
		provider: ai.Good,
		model:    modelGood,
		format:   formatText,
	}
}

// Creates a valid pull request review payload to be POSTed to the GitHub Rest API at /repos/{owner}/{repo}/pulls/{pull_number}/reviews.
//
// see https://docs.github.com/en/rest/pulls/reviews?apiVersion=2022-11-28#create-a-review-for-a-pull-request
func (ai *AI) GeneratePullRequestReview(number int, title, description, prDiff string) (*github.PullRequestReviewRequest, error) {
	notes, err := ai.generateReviewComments(number, title, description, prDiff)
	if err != nil {
		return nil, err
	}

	payload, _, err := ai.generateReviewBody(number, title, description, *notes)
	if err != nil {
		return nil, err
	}

	ai.fixProblemsWithPayload(prDiff, payload)

	return payload, nil
}

func (ai *AI) generateReviewComments(number int, title, description, prDiff string) (*string, error) {
	details := formatPullRequestDetails(number, title, description)
	message := fmt.Sprintf(reviewCommentsPrompt, details, ai.addPositionNumbersToDiff(prDiff))

	resp, err := ai.NewCompletion().Create(message)
	if err != nil {
		return nil, err
	}

	return &resp.Completion, nil
}

func (ai *AI) generateReviewBody(number int, title, description, notes string) (*github.PullRequestReviewRequest, *string, error) {
	details := formatPullRequestDetails(number, title, description)
	message := fmt.Sprintf(reviewPostBodyPrompt, details, notes)

	resp, err := ai.NewCompletion().Cheap().ReturnJSON().Create(message)
	if err != nil {
		return nil, nil, err
	}

	// OpenAI claims that the response will always be valid json when using the JSON response format
	var payload github.PullRequestReviewRequest
	err = json.Unmarshal([]byte(resp.Completion), &payload)
	if err != nil {
		return nil, nil, err
	}

	return &payload, &resp.Completion, nil
}

// Create a reply for thread of GitHub comments on a particular pull request hunk. The output of the
// string "noreply" indicates that no reply should be made (ie. the conversation has reached an end).
func (ai *AI) GenerateCommentReply(comment, hunk string, allComments []*github.PullRequestComment, name string) (*string, error) {
	thread := formatPullRequestComments(allComments)
	message := fmt.Sprintf(commentReplyPrompt, comment, hunk, name, thread)

	resp, err := ai.NewCompletion().Create(message)
	if err != nil {
		return nil, err
	}

	return &resp.Completion, nil
}

// Build a prompt snippet for the details of a pull request
func formatPullRequestDetails(number int, title string, description string) string {
	return fmt.Sprintf(
		"Pull Request #%d\nTitle: %s\n%s",
		number,
		title,
		description,
	)
}

// Build a prompt snippet for comments in a pull request thread. Output format looks like:
//
//	user1: comment
//	user2: comment
//	user1: comment
//	...
func formatPullRequestComments(comments []*github.PullRequestComment) string {
	result := ""
	for _, comment := range comments {
		user := comment.GetUser().GetLogin()
		body := comment.GetBody()
		result += fmt.Sprintf("%s: %s\n", user, body)
	}
	return result
}

// Check that the positions of the comments in a PR review are valid and fix any issues that
// are found.
// 1. Comments left a file not in the diff are removed.
// 2. Comments left on a position that is out of range are fixed to the max position in the diff.
// 3. Comments left on a diff with no hunk are removed.
func (ai *AI) fixProblemsWithPayload(diff string, body *github.PullRequestReviewRequest) {
	// Maps to keep track of the start line of a file's changes and the total count.
	// fileStarts maps file paths to their starting line in the diff.
	// positionCounts maps file paths to their maximum valid position.
	positionCounts := make(map[string]int)
	diffLines := strings.Split(diff, "\n")

	// line must start with "diff --git"
	getFileName := func(line string) string {
		pathParts := strings.Fields(line)
		if len(pathParts) > 2 {
			return pathParts[2][2:] // Removing the "a/" prefix.
		}
		return ""
	}

	// First, count the lines in each hunk.
	currFilePath := ""
	counter := 0
	shouldCount := false
	for _, line := range diffLines {
		if strings.HasPrefix(line, "diff --git") {
			// We have a new file to start a position counter for. Update the
			// positionCounts map with the data from the previous file before resetting.
			if currFilePath != "" {
				positionCounts[currFilePath] = counter
			}
			counter = 0
			shouldCount = false
			currFilePath = getFileName(line)
		} else if strings.HasPrefix(line, "@@") && currFilePath != "" {
			// Increase the counter if we have already started counting. The start
			// of neww hunks are valid positions. Otherwise, flip the shouldCount flag
			// but don't include the current position in the count.
			if !shouldCount {
				shouldCount = true
			} else {
				counter++
			}
		} else if currFilePath != "" && shouldCount {
			counter++
		}
	}

	// Add the last file.
	positionCounts[currFilePath] = counter

	// Now, check the comments against the counts.
	// Iterate backwards over the slice since the
	for i := len(body.Comments) - 1; i >= 0; i-- {
		comment := body.Comments[i]
		maxPos, exists := positionCounts[*comment.Path]
		if !exists {
			// Remove the comment.
			body.Comments = append(body.Comments[:i], body.Comments[i+1:]...)
		} else if maxPos == 0 {
			// This file has no diff hunk. As far as I can tell we can't leave a comment through the GitHub API.
			body.Comments = append(body.Comments[:i], body.Comments[i+1:]...)
		} else if *comment.Position > maxPos {
			// Fix the comment position so the review goes through
			*comment.Position = maxPos
		}
	}
}

// Add position numbers to the diff hunk for each file. This will enable an AI model to
// better identify the position in the diff that a comment should be placed on.
//
// The position value is the number of lines down from the first "@@" hunk header in the
// file you want to add a comment. The line just below the "@@" line is position 1, the
// next line is position 2, and so on. The position in the diff continues to increase
// through lines of whitespace and additional hunks until the beginning of a new file.
func (ai *AI) addPositionNumbersToDiff(diff string) string {
	diffLines := strings.Split(diff, "\n")

	addNumberToLine := func(num int, line string) string {
		return fmt.Sprintf("%d %s", num, line)
	}

	counter := 0
	shouldCount := false
	for i, line := range diffLines {
		if strings.HasPrefix(line, "diff --git") {
			counter = 0
			shouldCount = false
		} else if strings.HasPrefix(line, "@@") {
			if !shouldCount {
				shouldCount = true
			} else {
				counter++
				diffLines[i] = addNumberToLine(counter, line)
			}
		} else if shouldCount {
			counter++
			diffLines[i] = addNumberToLine(counter, line)
		}
	}

	return strings.Join(diffLines, "\n")
}
