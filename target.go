package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// target identifies whose usage to report. Repo is empty for owner-wide usage.
type target struct {
	Owner string
	Repo  string
}

func (t target) String() string {
	if t.Repo == "" {
		return t.Owner
	}
	return t.Owner + "/" + t.Repo
}

var (
	ownerPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$`)
	repoPattern  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func parseTarget(s string) (target, error) {
	owner, repo, hasRepo := strings.Cut(s, "/")

	if !ownerPattern.MatchString(owner) {
		return target{}, fmt.Errorf("invalid owner %q: expected OWNER or OWNER/REPO", s)
	}
	if hasRepo && (!repoPattern.MatchString(repo) || repo == "." || repo == "..") {
		return target{}, fmt.Errorf("invalid repository %q: expected OWNER or OWNER/REPO", s)
	}

	return target{Owner: owner, Repo: repo}, nil
}

// period is a billing month.
type period struct {
	Year  int
	Month time.Month
}

func (p period) String() string {
	return time.Date(p.Year, p.Month, 1, 0, 0, 0, 0, time.UTC).Format("Jan 2006")
}

// parseMonth parses YYYY-MM, defaulting to the month of now when empty.
func parseMonth(s string, now time.Time) (period, error) {
	if s == "" {
		return period{Year: now.Year(), Month: now.Month()}, nil
	}

	t, err := time.Parse("2006-01", s)
	if err != nil {
		return period{}, fmt.Errorf("invalid month %q: expected YYYY-MM", s)
	}
	return period{Year: t.Year(), Month: t.Month()}, nil
}
