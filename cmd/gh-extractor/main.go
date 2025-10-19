package main

import (
	"log/slog"
	"os"

	"gh-extractor/internal/clients"
	"gh-extractor/internal/repository"
	"gh-extractor/internal/usecase"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get current working directory", "error", err)
		os.Exit(1)
	}

	// Initialize dependencies
	githubClient := clients.NewGitHubClient()
	fileRepo := repository.NewFileRepository(cwd + "/.data")
	extractorUseCase := usecase.NewExtractorUseCase(githubClient, fileRepo, logger)

	// Execute extraction
	logger.Info("Starting GitHub activity extraction")
	if err := extractorUseCase.Extract(); err != nil {
		logger.Error("Extraction failed", "error", err)
		os.Exit(1)
	}

	logger.Info("All done!")
}
