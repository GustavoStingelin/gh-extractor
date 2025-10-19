package usecase

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"gh-extractor/internal/clients"
	"gh-extractor/internal/repository"
)

func TestNewExtractorUseCase(t *testing.T) {
	// Setup
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	githubClient := clients.NewGitHubClient()
	fileRepo := repository.NewFileRepository(t.TempDir())

	// Execute
	useCase := NewExtractorUseCase(githubClient, fileRepo, logger)

	// Assert
	if useCase == nil {
		t.Fatal("Expected non-nil ExtractorUseCase")
	}

	if useCase.githubClient == nil {
		t.Error("Expected non-nil githubClient")
	}

	if useCase.fileRepo == nil {
		t.Error("Expected non-nil fileRepo")
	}

	if useCase.logger == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestDateCalculation(t *testing.T) {
	// Test that we can calculate 3 months ago correctly
	now := time.Date(2025, 10, 18, 0, 0, 0, 0, time.UTC)
	threeMonthsAgo := now.AddDate(0, -3, 0)

	expected := time.Date(2025, 7, 18, 0, 0, 0, 0, time.UTC)

	if !threeMonthsAgo.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, threeMonthsAgo)
	}
}
