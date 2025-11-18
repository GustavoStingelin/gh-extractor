default:
	@just --list

# Run extractor to download/update GitHub activity data
extract:
	@echo "Running GitHub activity extractor..."
	go run ./cmd/gh-extractor

# Generate monthly summary: `just summary 10 2025`
summary month year:
	@echo "Generating summary for {{month}}/{{year}}..."
	go run ./cmd/gh-summary --month {{month}} --year {{year}}

# Run tests
test:
	go test ./...
