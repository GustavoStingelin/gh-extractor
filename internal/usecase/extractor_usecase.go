package usecase

import (
	"fmt"
	"log/slog"
	"time"

	"gh-extractor/internal/clients"
	"gh-extractor/internal/repository"
)

type ExtractorUseCase struct {
	githubClient *clients.GitHubClient
	fileRepo     *repository.FileRepository
	logger       *slog.Logger
}

func NewExtractorUseCase(githubClient *clients.GitHubClient, fileRepo *repository.FileRepository, logger *slog.Logger) *ExtractorUseCase {
	return &ExtractorUseCase{
		githubClient: githubClient,
		fileRepo:     fileRepo,
		logger:       logger,
	}
}

// Extract orchestrates the entire extraction process
func (u *ExtractorUseCase) Extract() error {
	// Ensure base directory exists
	if err := u.fileRepo.EnsureBaseDirectory(); err != nil {
		return fmt.Errorf("failed to ensure base directory: %w", err)
	}

	// Calculate date 3 months ago
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	u.logger.Info("Extracting GitHub activity", "since", threeMonthsAgo.Format("2006-01-02"))

	// Extract authored PRs
	if err := u.extractAuthoredPRs(threeMonthsAgo); err != nil {
		return fmt.Errorf("failed to extract authored PRs: %w", err)
	}

	// Extract reviewed PRs
	if err := u.extractReviewedPRs(threeMonthsAgo); err != nil {
		return fmt.Errorf("failed to extract reviewed PRs: %w", err)
	}

	u.logger.Info("Extraction completed successfully")
	return nil
}

// extractAuthoredPRs extracts PRs authored by the user
func (u *ExtractorUseCase) extractAuthoredPRs(since time.Time) error {
	u.logger.Info("Searching for authored PRs")

	prs, err := u.githubClient.SearchAuthoredPRs(since)
	if err != nil {
		return err
	}

	u.logger.Info("Found authored PRs", "count", len(prs))

	for i, pr := range prs {
		u.logger.Info("Processing authored PR",
			"progress", fmt.Sprintf("%d/%d", i+1, len(prs)),
			"repo", pr.Repository.NameWithOwner,
			"number", pr.Number,
			"title", pr.Title)

		if err := u.processPR(pr.Repository.NameWithOwner, pr.Number, "pr"); err != nil {
			u.logger.Warn("Failed to process PR",
				"repo", pr.Repository.NameWithOwner,
				"number", pr.Number,
				"error", err)
			continue
		}
	}

	return nil
}

// extractReviewedPRs extracts PRs reviewed by the user
func (u *ExtractorUseCase) extractReviewedPRs(since time.Time) error {
	u.logger.Info("Searching for reviewed PRs")

	prs, err := u.githubClient.SearchReviewedPRs(since)
	if err != nil {
		return err
	}

	u.logger.Info("Found reviewed PRs", "count", len(prs))

	for i, pr := range prs {
		u.logger.Info("Processing reviewed PR",
			"progress", fmt.Sprintf("%d/%d", i+1, len(prs)),
			"repo", pr.Repository.NameWithOwner,
			"number", pr.Number,
			"title", pr.Title)

		if err := u.processPR(pr.Repository.NameWithOwner, pr.Number, "review"); err != nil {
			u.logger.Warn("Failed to process PR",
				"repo", pr.Repository.NameWithOwner,
				"number", pr.Number,
				"error", err)
			continue
		}
	}

	return nil
}

// processPR fetches details and saves a PR
func (u *ExtractorUseCase) processPR(repo string, number int, prType string) error {
	// Fetch PR details
	prDetails, err := u.githubClient.GetPRDetails(repo, number)
	if err != nil {
		return fmt.Errorf("failed to get PR details: %w", err)
	}

	// Save PR data
	if err := u.fileRepo.SavePRData(prDetails, prType); err != nil {
		return fmt.Errorf("failed to save PR data: %w", err)
	}

	// Fetch and save PR diff
	diff, err := u.githubClient.GetPRDiff(repo, number)
	if err != nil {
		return fmt.Errorf("failed to get PR diff: %w", err)
	}

	if err := u.fileRepo.SaveDiff(prDetails, diff, prType); err != nil {
		return fmt.Errorf("failed to save diff: %w", err)
	}

	return nil
}
