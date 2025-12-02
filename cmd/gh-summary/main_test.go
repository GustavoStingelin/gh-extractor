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
