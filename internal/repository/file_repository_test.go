package repository

import (
	"testing"
	"time"

	"gh-extractor/internal/clients"
)

func TestPRStatusMissingData(t *testing.T) {
	repo := NewFileRepository(t.TempDir())

	exists, upToDate, err := repo.PRStatus("org/repo", 1, "pr", time.Now(), "open")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatalf("expected exists=false when files are missing")
	}
	if upToDate {
		t.Fatalf("expected upToDate=false when files are missing")
	}
}

func TestPRStatusDetectsStaleData(t *testing.T) {
	repo := NewFileRepository(t.TempDir())
	name := "org/repo"
	number := 2
	localUpdated := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	remoteUpdated := localUpdated.Add(24 * time.Hour)

	pr := &clients.PullRequest{
		Number:    number,
		UpdatedAt: localUpdated,
		Repository: clients.Repository{
			NameWithOwner: name,
		},
	}

	if err := repo.SavePRData(pr, "pr"); err != nil {
		t.Fatalf("failed to save PR data: %v", err)
	}
	if err := repo.SaveDiff(pr, "diff", "pr"); err != nil {
		t.Fatalf("failed to save diff: %v", err)
	}

	exists, upToDate, err := repo.PRStatus(name, number, "pr", remoteUpdated, "open")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatalf("expected exists=true after saving data")
	}
	if upToDate {
		t.Fatalf("expected upToDate=false when remote data is newer")
	}
}

func TestPRStatusUpToDate(t *testing.T) {
	repo := NewFileRepository(t.TempDir())
	name := "org/repo"
	number := 3
	remoteUpdated := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

	pr := &clients.PullRequest{
		Number:    number,
		UpdatedAt: remoteUpdated,
		Repository: clients.Repository{
			NameWithOwner: name,
		},
	}

	if err := repo.SavePRData(pr, "review"); err != nil {
		t.Fatalf("failed to save PR data: %v", err)
	}
	if err := repo.SaveDiff(pr, "diff", "review"); err != nil {
		t.Fatalf("failed to save diff: %v", err)
	}

	exists, upToDate, err := repo.PRStatus(name, number, "review", remoteUpdated, "closed")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatalf("expected exists=true after saving data")
	}
	if !upToDate {
		t.Fatalf("expected upToDate=true when local data is current")
	}
}

func TestPRStatusDifferentState(t *testing.T) {
	repo := NewFileRepository(t.TempDir())
	name := "org/repo"
	number := 4
	updated := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	pr := &clients.PullRequest{
		Number:    number,
		State:     "open",
		UpdatedAt: updated,
		Repository: clients.Repository{
			NameWithOwner: name,
		},
	}

	if err := repo.SavePRData(pr, "pr"); err != nil {
		t.Fatalf("failed to save PR data: %v", err)
	}
	if err := repo.SaveDiff(pr, "diff", "pr"); err != nil {
		t.Fatalf("failed to save diff: %v", err)
	}

	exists, upToDate, err := repo.PRStatus(name, number, "pr", updated, "merged")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatalf("expected exists=true after saving data")
	}
	if upToDate {
		t.Fatalf("expected upToDate=false when remote state differs")
	}
}
