# GitHub Activity Extractor

A Go CLI tool that extracts your GitHub activity from the last 3 months using the `gh` CLI and saves it in a structured YAML format.

## Development

Extract data with verbose logging:

```bash
go run ./cmd/gh-extractor
```

Generate a monthly summary (for example, October 2025):

```bash
go run ./cmd/gh-summary --month 10 --year 2025
```

This will write a Markdown file to:

- `.data/summary/2025-10.md`

Run tests:

```bash
go test ./...
```
