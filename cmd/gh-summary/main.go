package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gh-extractor/internal/clients"

	"gopkg.in/yaml.v3"
)

type authoredPRSummary struct {
	Repository       string
	Number           int
	Title            string
	URL              string
	CreatedAt        time.Time
	State            string
	CommentsMade     int
	CommentsReceived int
}

type reviewedPRSummary struct {
	Repository string
	Number     int
	Title      string
	URL        string
	ReviewDate time.Time
	Outcome    string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	var month, year int
	flag.IntVar(&month, "month", 0, "Month to summarize (1-12). Defaults to current month.")
	flag.IntVar(&year, "year", 0, "Year to summarize. Defaults to current year.")
	flag.Parse()

	now := time.Now()
	if month == 0 {
		month = int(now.Month())
	}
	if year == 0 {
		year = now.Year()
	}

	if month < 1 || month > 12 {
		logger.Error("Invalid month value", "month", month)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get current working directory", "error", err)
		os.Exit(1)
	}

	baseDir := filepath.Join(cwd, ".data")

	userLogin, err := detectUserLogin(baseDir)
	if err != nil {
		logger.Error("Failed to detect user login", "error", err)
		os.Exit(1)
	}

	logger.Info("Generating GitHub monthly summary",
		"month", month,
		"year", year,
		"user", userLogin)

	authoredCreated, authoredUpdated, err := summarizeAuthoredPRs(baseDir, userLogin, month, year)
	if err != nil {
		logger.Error("Failed to summarize authored PRs", "error", err)
		os.Exit(1)
	}

	reviewed, err := summarizeReviewedPRs(baseDir, userLogin, month, year)
	if err != nil {
		logger.Error("Failed to summarize reviewed PRs", "error", err)
		os.Exit(1)
	}

	mdPath, err := writeMarkdownSummary(authoredCreated, authoredUpdated, reviewed, month, year, baseDir)
	if err != nil {
		logger.Error("Failed to write markdown summary", "error", err)
	} else {
		logger.Info("Markdown summary written", "path", mdPath)
	}
}

func detectUserLogin(baseDir string) (string, error) {
	prDir := filepath.Join(baseDir, "pr")

	var login string
	err := filepath.WalkDir(prDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		pr, err := loadPullRequestFromYAML(path)
		if err != nil {
			return err
		}
		if pr.Author.Login != "" {
			login = pr.Author.Login
		}
		return fs.SkipAll
	})

	if err != nil {
		return "", err
	}
	if login == "" {
		return "", fmt.Errorf("could not detect user login from %s", prDir)
	}

	return login, nil
}

func summarizeAuthoredPRs(baseDir, userLogin string, month, year int) ([]authoredPRSummary, []authoredPRSummary, error) {
	prDir := filepath.Join(baseDir, "pr")
	var createdThisMonth []authoredPRSummary
	var previousWithActivity []authoredPRSummary

	err := filepath.WalkDir(prDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		pr, err := loadPullRequestFromYAML(path)
		if err != nil {
			// Best-effort: skip invalid files
			return nil
		}

		// Ensure this is a PR authored by the user
		if pr.Author.Login != userLogin {
			return nil
		}

		made, received := countCommentsForAuthoredPR(pr, userLogin, month, year)

		createdInMonth := isInMonthYear(pr.CreatedAt, month, year)
		hadActivity := made > 0 || received > 0

		if !createdInMonth && !hadActivity {
			return nil
		}

		summary := authoredPRSummary{
			Repository:       pr.Repository.NameWithOwner,
			Number:           pr.Number,
			Title:            pr.Title,
			URL:              pr.URL,
			CreatedAt:        pr.CreatedAt,
			State:            pr.State,
			CommentsMade:     made,
			CommentsReceived: received,
		}

		if createdInMonth {
			createdThisMonth = append(createdThisMonth, summary)
		} else {
			previousWithActivity = append(previousWithActivity, summary)
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}

	sort.Slice(createdThisMonth, func(i, j int) bool {
		return createdThisMonth[i].CreatedAt.Before(createdThisMonth[j].CreatedAt)
	})

	sort.Slice(previousWithActivity, func(i, j int) bool {
		return previousWithActivity[i].CreatedAt.Before(previousWithActivity[j].CreatedAt)
	})

	return createdThisMonth, previousWithActivity, nil
}

func summarizeReviewedPRs(baseDir, userLogin string, month, year int) ([]reviewedPRSummary, error) {
	reviewDir := filepath.Join(baseDir, "review")
	byPRAndDay := make(map[string]*reviewedPRSummary)

	err := filepath.WalkDir(reviewDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		pr, err := loadPullRequestFromYAML(path)
		if err != nil {
			// Best-effort: skip invalid files
			return nil
		}

		for _, review := range pr.Reviews {
			if review.Author.Login != userLogin {
				continue
			}
			if !isInMonthYear(review.SubmittedAt, month, year) {
				continue
			}

			day := review.SubmittedAt.Format("2006-01-02")
			key := fmt.Sprintf("%s#%d@%s", pr.Repository.NameWithOwner, pr.Number, day)
			outcome := normalizeOutcome(review.State)

			if existing, ok := byPRAndDay[key]; ok {
				// Keep earliest time in the day for ordering
				if review.SubmittedAt.Before(existing.ReviewDate) {
					existing.ReviewDate = review.SubmittedAt
				}
				existing.Outcome = mergeOutcome(existing.Outcome, outcome)
			} else {
				byPRAndDay[key] = &reviewedPRSummary{
					Repository: pr.Repository.NameWithOwner,
					Number:     pr.Number,
					Title:      pr.Title,
					URL:        pr.URL,
					ReviewDate: review.SubmittedAt,
					Outcome:    outcome,
				}
			}
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var summaries []reviewedPRSummary
	for _, s := range byPRAndDay {
		summaries = append(summaries, *s)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ReviewDate.Before(summaries[j].ReviewDate)
	})

	return summaries, nil
}

func loadPullRequestFromYAML(path string) (*clients.PullRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pr clients.PullRequest
	if err := yaml.Unmarshal(data, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func countCommentsForAuthoredPR(pr *clients.PullRequest, userLogin string, month, year int) (made, received int) {
	for _, c := range pr.Comments {
		if !isInMonthYear(c.CreatedAt, month, year) {
			continue
		}
		if c.Author.Login == userLogin {
			made++
		} else {
			received++
		}
	}

	for _, review := range pr.Reviews {
		for _, rc := range review.ReviewComments {
			if !isInMonthYear(rc.CreatedAt, month, year) {
				continue
			}
			if rc.User.Login == userLogin {
				made++
			} else {
				received++
			}
		}
	}

	return made, received
}

func isInMonthYear(t time.Time, month, year int) bool {
	if t.IsZero() {
		return false
	}
	return t.Year() == year && int(t.Month()) == month
}

func writeMarkdownSummary(authoredCreated, authoredUpdated []authoredPRSummary, reviewed []reviewedPRSummary, month, year int, baseDir string) (string, error) {
	monthName := time.Month(month).String()

	summaryDir := filepath.Join(baseDir, "summary")
	if err := os.MkdirAll(summaryDir, 0o755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%04d-%02d.md", year, month)
	path := filepath.Join(summaryDir, filename)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()

	fmt.Fprintf(w, "Resumo %s/%d\n\n", monthName, year)

	fmt.Fprintln(w, "PRs criados")
	fmt.Fprintln(w)
	if len(authoredCreated) == 0 {
		fmt.Fprintln(w, "- Nenhum PR criado.")
	} else {
		for _, pr := range authoredCreated {
			fmt.Fprintf(
				w,
				"- %s#%d — %s (%s) — criado: %s — status: %s — meus comentários (mês): %d — comentários recebidos (mês): %d\n",
				pr.Repository,
				pr.Number,
				pr.Title,
				pr.URL,
				pr.CreatedAt.Format("2006-01-02"),
				pr.State,
				pr.CommentsMade,
				pr.CommentsReceived,
			)
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "PRs antigos com atividade")
	fmt.Fprintln(w)
	if len(authoredUpdated) == 0 {
		fmt.Fprintln(w, "- Nenhum PR antigo com atividade.")
	} else {
		for _, pr := range authoredUpdated {
			fmt.Fprintf(
				w,
				"- %s#%d — %s (%s) — criado: %s — status: %s — meus comentários (mês): %d — comentários recebidos (mês): %d\n",
				pr.Repository,
				pr.Number,
				pr.Title,
				pr.URL,
				pr.CreatedAt.Format("2006-01-02"),
				pr.State,
				pr.CommentsMade,
				pr.CommentsReceived,
			)
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "PRs revisados")
	fmt.Fprintln(w)
	if len(reviewed) == 0 {
		fmt.Fprintln(w, "- Nenhum PR revisado.")
	} else {
		for _, pr := range reviewed {
			outcome := ""
			if pr.Outcome != "" {
				outcome = fmt.Sprintf(" — outcome: %s", pr.Outcome)
			}
			fmt.Fprintf(
				w,
				"- %s#%d — %s (%s) — revisão: %s%s\n",
				pr.Repository,
				pr.Number,
				pr.Title,
				pr.URL,
				pr.ReviewDate.Format("2006-01-02"),
				outcome,
			)
		}
	}

	return path, nil
}

func normalizeOutcome(state string) string {
	switch state {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes requested"
	case "COMMENTED":
		return "commented"
	default:
		return ""
	}
}

func mergeOutcome(current, new string) string {
	if new == "" {
		return current
	}
	// Priority: changes requested > approved > commented
	if current == "" {
		return new
	}
	if current == "changes requested" || new == "changes requested" {
		return "changes requested"
	}
	if current == "approved" || new == "approved" {
		return "approved"
	}
	return "commented"
}
