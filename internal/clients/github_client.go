package clients

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type GitHubClient struct{}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{}
}

// Repository represents a GitHub repository
type Repository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

// Author represents a GitHub user
type Author struct {
	ID    interface{} `json:"id"` // Can be string or number depending on API
	Login string      `json:"login"`
	Name  string      `json:"name"`
	IsBot bool        `json:"is_bot"`
}

// Comment represents a comment on a PR
type Comment struct {
	ID        string    `json:"id"`
	Author    Author    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ReviewComment represents a code review comment on a specific line
type ReviewComment struct {
	ID        int       `json:"id"`
	NodeID    string    `json:"node_id"`
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	StartLine int       `json:"start_line"`
	Side      string    `json:"side"`
	User      Author    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ReviewID  int       `json:"pull_request_review_id"`
}

// Review represents a PR review
type Review struct {
	ID             string          `json:"id"`
	Author         Author          `json:"author"`
	State          string          `json:"state"`
	Body           string          `json:"body"`
	SubmittedAt    time.Time       `json:"submittedAt"`
	ReviewComments []ReviewComment `json:"reviewComments,omitempty"`
}

// PullRequest represents a GitHub pull request with all details
type PullRequest struct {
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	State      string     `json:"state"`
	URL        string     `json:"url"`
	Author     Author     `json:"author"`
	Repository Repository `json:"repository"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	Comments   []Comment  `json:"comments"`
	Reviews    []Review   `json:"reviews"`
}

// PRSearchResult represents a simplified PR from search results
type PRSearchResult struct {
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	State      string     `json:"state"`
	URL        string     `json:"url"`
	Repository Repository `json:"repository"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// SearchAuthoredPRs searches for PRs authored by the authenticated user since the given date
func (c *GitHubClient) SearchAuthoredPRs(since time.Time) ([]PRSearchResult, error) {
	queryArgs := []string{
		"is:pr",
		"author:@me",
		fmt.Sprintf("created:>=%s", since.Format("2006-01-02")),
	}
	return c.searchPRs(queryArgs)
}

// SearchReviewedPRs searches for PRs reviewed by the authenticated user since the given date
func (c *GitHubClient) SearchReviewedPRs(since time.Time) ([]PRSearchResult, error) {
	// Note: We cannot use -author:@me as a separate arg because the leading dash
	// makes it look like a flag. We'll need to filter out authored PRs later.
	queryArgs := []string{
		"is:pr",
		"reviewed-by:@me",
		fmt.Sprintf("created:>=%s", since.Format("2006-01-02")),
	}
	return c.searchPRs(queryArgs)
}

// searchPRs executes a PR search query
func (c *GitHubClient) searchPRs(queryArgs []string) ([]PRSearchResult, error) {
	args := []string{"search", "prs", "--limit", "1000", "--json", "number,title,state,url,repository,createdAt"}
	args = append(args, queryArgs...)

	cmd := exec.Command("gh", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to search PRs: %w (output: %s)", err, string(output))
	}
	var results []PRSearchResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return results, nil
}

// APIReview represents a review from the GitHub API (with numeric ID)
type APIReview struct {
	ID          int       `json:"id"`
	NodeID      string    `json:"node_id"`
	User        Author    `json:"user"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// GetPRDetails fetches detailed information about a specific PR
func (c *GitHubClient) GetPRDetails(repo string, number int) (*PullRequest, error) {
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", number),
		"--repo", repo,
		"--json", "number,title,body,state,url,author,createdAt,updatedAt,comments,reviews")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR details: %w (output: %s)", err, string(output))
	}

	var pr PullRequest
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR details: %w", err)
	}

	// Populate repository from the repo parameter
	pr.Repository = Repository{
		NameWithOwner: repo,
	}

	// Fetch reviews via API to get numeric IDs
	apiReviews, err := c.getAPIReviews(repo, number)
	if err != nil {
		fmt.Printf("Warning: failed to fetch API reviews for %s#%d: %v\n", repo, number, err)
		return &pr, nil
	}

	// Fetch review comments and associate them with reviews
	reviewComments, err := c.GetPRReviewComments(repo, number)
	if err != nil {
		fmt.Printf("Warning: failed to fetch review comments for %s#%d: %v\n", repo, number, err)
		return &pr, nil
	}

	// Associate comments with reviews using node_id matching
	c.associateReviewComments(&pr, apiReviews, reviewComments)

	return &pr, nil
}

// getAPIReviews fetches reviews via GitHub API to get numeric IDs
func (c *GitHubClient) getAPIReviews(repo string, number int) ([]APIReview, error) {
	cmd := exec.Command("gh", "api",
		fmt.Sprintf("repos/%s/pulls/%d/reviews", repo, number),
		"--paginate")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get API reviews: %w (output: %s)", err, string(output))
	}

	var reviews []APIReview
	if err := json.Unmarshal(output, &reviews); err != nil {
		return nil, fmt.Errorf("failed to parse API reviews: %w", err)
	}

	return reviews, nil
}

// GetPRReviewComments fetches all review comments for a PR
func (c *GitHubClient) GetPRReviewComments(repo string, number int) ([]ReviewComment, error) {
	cmd := exec.Command("gh", "api",
		fmt.Sprintf("repos/%s/pulls/%d/comments", repo, number),
		"--paginate")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get review comments: %w (output: %s)", err, string(output))
	}

	var comments []ReviewComment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}

	return comments, nil
}

// associateReviewComments groups review comments by review ID and adds them to reviews
func (c *GitHubClient) associateReviewComments(pr *PullRequest, apiReviews []APIReview, reviewComments []ReviewComment) {
	// Build a map of node_id to numeric ID
	nodeIDToNumericID := make(map[string]int)
	for _, apiReview := range apiReviews {
		nodeIDToNumericID[apiReview.NodeID] = apiReview.ID
	}

	// Build a map of review numeric ID to review comments
	commentsByReviewID := make(map[int][]ReviewComment)
	for _, comment := range reviewComments {
		if comment.ReviewID != 0 {
			commentsByReviewID[comment.ReviewID] = append(commentsByReviewID[comment.ReviewID], comment)
		}
	}

	// Associate comments with reviews
	for i := range pr.Reviews {
		review := &pr.Reviews[i]
		if numericID, ok := nodeIDToNumericID[review.ID]; ok {
			if comments, found := commentsByReviewID[numericID]; found {
				review.ReviewComments = comments
			}
		}
	}
}

// GetPRDiff fetches the complete diff for a specific PR (all commits)
func (c *GitHubClient) GetPRDiff(repo string, number int) (string, error) {
	// Using the GitHub API to get the full PR patch ensures we get all commits
	cmd := exec.Command("gh", "api",
		fmt.Sprintf("repos/%s/pulls/%d", repo, number),
		"-H", "Accept: application/vnd.github.v3.diff")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}
