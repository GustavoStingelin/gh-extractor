package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gh-extractor/internal/clients"

	"gopkg.in/yaml.v3"
)

type FileRepository struct {
	baseDir string
}

func NewFileRepository(baseDir string) *FileRepository {
	return &FileRepository{
		baseDir: baseDir,
	}
}

// SavePRData saves PR data as YAML in the appropriate directory structure
func (r *FileRepository) SavePRData(pr *clients.PullRequest, prType string) error {
	// Parse org and repo from nameWithOwner (e.g., "btcsuite/btcwallet")
	parts := strings.Split(pr.Repository.NameWithOwner, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository name format: %s", pr.Repository.NameWithOwner)
	}
	org, repo := parts[0], parts[1]

	// Create directory path: .data/{prType}/{org}/{repo}
	dirPath := filepath.Join(r.baseDir, prType, org, repo)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Marshal PR data to YAML
	yamlData, err := yaml.Marshal(pr)
	if err != nil {
		return fmt.Errorf("failed to marshal PR data to YAML: %w", err)
	}

	// Write YAML file: {number}.yaml
	yamlPath := filepath.Join(dirPath, fmt.Sprintf("%d.yaml", pr.Number))
	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file %s: %w", yamlPath, err)
	}

	return nil
}

// SaveDiff saves PR diff in the appropriate directory structure
func (r *FileRepository) SaveDiff(pr *clients.PullRequest, diff string, prType string) error {
	// Parse org and repo from nameWithOwner
	parts := strings.Split(pr.Repository.NameWithOwner, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository name format: %s", pr.Repository.NameWithOwner)
	}
	org, repo := parts[0], parts[1]

	// Create directory path: .data/{prType}/{org}/{repo}
	dirPath := filepath.Join(r.baseDir, prType, org, repo)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Write diff file: {number}.diff
	diffPath := filepath.Join(dirPath, fmt.Sprintf("%d.diff", pr.Number))
	if err := os.WriteFile(diffPath, []byte(diff), 0644); err != nil {
		return fmt.Errorf("failed to write diff file %s: %w", diffPath, err)
	}

	return nil
}

// EnsureBaseDirectory ensures the base .data directory exists
func (r *FileRepository) EnsureBaseDirectory() error {
	if err := os.MkdirAll(r.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", r.baseDir, err)
	}
	return nil
}

// PRExists checks if both YAML and diff files exist for a PR
func (r *FileRepository) PRExists(nameWithOwner string, number int, prType string) bool {
	// Parse org and repo from nameWithOwner
	parts := strings.Split(nameWithOwner, "/")
	if len(parts) != 2 {
		return false
	}
	org, repo := parts[0], parts[1]

	// Check for YAML file
	yamlPath := filepath.Join(r.baseDir, prType, org, repo, fmt.Sprintf("%d.yaml", number))
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return false
	}

	// Check for diff file
	diffPath := filepath.Join(r.baseDir, prType, org, repo, fmt.Sprintf("%d.diff", number))
	if _, err := os.Stat(diffPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// PRStatus reports whether a PR has already been downloaded and whether the local data
// is up-to-date compared to the remote updatedAt timestamp and state. Missing files,
// older local data, or differing state return upToDate=false.
func (r *FileRepository) PRStatus(nameWithOwner string, number int, prType string, remoteUpdatedAt time.Time, remoteState string) (exists bool, upToDate bool, err error) {
	parts := strings.Split(nameWithOwner, "/")
	if len(parts) != 2 {
		return false, false, fmt.Errorf("invalid repository name format: %s", nameWithOwner)
	}

	org, repo := parts[0], parts[1]
	yamlPath := filepath.Join(r.baseDir, prType, org, repo, fmt.Sprintf("%d.yaml", number))
	diffPath := filepath.Join(r.baseDir, prType, org, repo, fmt.Sprintf("%d.diff", number))

	if _, err := os.Stat(yamlPath); err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}

	if _, err := os.Stat(diffPath); err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}

	// At this point both files exist.
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return true, false, fmt.Errorf("failed to read YAML file %s: %w", yamlPath, err)
	}

	var pr clients.PullRequest
	if err := yaml.Unmarshal(data, &pr); err != nil {
		return true, false, fmt.Errorf("failed to parse YAML file %s: %w", yamlPath, err)
	}

	if pr.UpdatedAt.IsZero() || remoteUpdatedAt.IsZero() {
		return true, false, nil
	}

	if remoteUpdatedAt.After(pr.UpdatedAt) {
		return true, false, nil
	}

	if pr.State != "" && remoteState != "" && !strings.EqualFold(pr.State, remoteState) {
		return true, false, nil
	}

	return true, true, nil
}
