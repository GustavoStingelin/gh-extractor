package main

import (
	"testing"
	"time"

	"gh-extractor/internal/clients"
)

func TestCountCommentsForAuthoredPRCountsReviewBodies(t *testing.T) {
	user := "author"
	month, year := 11, 2025

	pr := &clients.PullRequest{
		Comments: []clients.Comment{},
		Reviews: []clients.Review{
			{
				Author:      clients.Author{Login: "reviewer"},
				Body:        "Looks good",
				SubmittedAt: time.Date(2025, 11, 27, 0, 0, 0, 0, time.UTC),
			},
			{
				Author:      clients.Author{Login: user},
				Body:        "Follow-up",
				SubmittedAt: time.Date(2025, 11, 28, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	made, received := countCommentsForAuthoredPR(pr, user, month, year)
	if made != 1 {
		t.Fatalf("expected 1 comment made from review body, got %d", made)
	}
	if received != 1 {
		t.Fatalf("expected 1 comment received from review body, got %d", received)
	}
}

func TestCountCommentsForAuthoredPRSkipsEmptyReviewBodies(t *testing.T) {
	user := "author"
	month, year := 11, 2025

	pr := &clients.PullRequest{
		Reviews: []clients.Review{
			{
				Author:      clients.Author{Login: "reviewer"},
				Body:        "",
				SubmittedAt: time.Date(2025, 11, 27, 0, 0, 0, 0, time.UTC),
				ReviewComments: []clients.ReviewComment{
					{
						User:      clients.Author{Login: "reviewer"},
						CreatedAt: time.Date(2025, 11, 27, 0, 0, 0, 0, time.UTC),
					},
				},
			},
		},
	}

	made, received := countCommentsForAuthoredPR(pr, user, month, year)
	if made != 0 {
		t.Fatalf("expected 0 comments made, got %d", made)
	}
	if received != 1 {
		t.Fatalf("expected 1 comment received from inline review comment, got %d", received)
	}
}

func TestGroupReviewedPRs(t *testing.T) {
	reviewed := []reviewedPRSummary{
		{
			Repository: "org/repo",
			Number:     1,
			Title:      "title",
			URL:        "url",
			ReviewDate: time.Date(2025, 11, 27, 0, 0, 0, 0, time.UTC),
			Outcome:    "commented",
		},
		{
			Repository: "org/other",
			Number:     2,
			Title:      "title2",
			URL:        "url2",
			ReviewDate: time.Date(2025, 11, 28, 0, 0, 0, 0, time.UTC),
			Outcome:    "approved",
		},
		{
			Repository: "org/repo",
			Number:     1,
			Title:      "title",
			URL:        "url",
			ReviewDate: time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC),
			Outcome:    "approved",
		},
	}

	order, grouped := groupReviewedPRs(reviewed)

	if len(order) != 2 {
		t.Fatalf("expected 2 PRs in order, got %d", len(order))
	}
	if order[0] != "org/repo#1" || order[1] != "org/other#2" {
		t.Fatalf("unexpected order: %v", order)
	}

	if len(grouped["org/repo#1"]) != 2 {
		t.Fatalf("expected 2 entries for org/repo#1, got %d", len(grouped["org/repo#1"]))
	}
	if grouped["org/repo#1"][0].Outcome != "commented" || grouped["org/repo#1"][1].Outcome != "approved" {
		t.Fatalf("unexpected outcomes for org/repo#1: %v", grouped["org/repo#1"])
	}
}
