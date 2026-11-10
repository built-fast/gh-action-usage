// Command gh-action-usage is a GitHub CLI extension that reports GitHub
// Actions billing usage for an organization, user, or repository.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

const usageText = `Usage: gh action-usage [flags] [OWNER | OWNER/REPO]

Show GitHub Actions billing usage for a month.

If no target is given, the repository in the current directory is used.

Flags:
  -m, --month YYYY-MM   Month to report on (default: current month)
      --by-repo         Break down usage by repository (for an OWNER;
                        defaults to the current repo's owner)
  -h, --help            Show this help
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("gh action-usage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usageText) }

	var month string
	fs.StringVar(&month, "month", "", "")
	fs.StringVar(&month, "m", "", "")

	var byRepo bool
	fs.BoolVar(&byRepo, "by-repo", false, "")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return errors.New("too many arguments")
	}

	period, err := parseMonth(month, time.Now())
	if err != nil {
		return err
	}

	tgt, err := resolveTarget(fs.Arg(0))
	if err != nil {
		return err
	}

	if byRepo && tgt.Repo != "" {
		if fs.Arg(0) != "" {
			return fmt.Errorf("--by-repo requires an OWNER, not a repository (got %s)", tgt)
		}
		// Inferred from the current directory: report on the repo's owner.
		tgt.Repo = ""
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}

	if byRepo {
		usage, err := fetchByRepo(client, tgt.Owner, period)
		if err != nil {
			return err
		}
		return renderByRepo(stdout, tgt, period, usage)
	}

	summary, err := fetchSummary(client, tgt, period)
	if err != nil {
		return err
	}

	return render(stdout, tgt, period, summary)
}

// resolveTarget parses an explicit OWNER or OWNER/REPO argument, or falls
// back to the repository for the current directory.
func resolveTarget(arg string) (target, error) {
	if arg != "" {
		return parseTarget(arg)
	}

	repo, err := repository.Current()
	if err != nil {
		return target{}, errors.New("not in a GitHub repository; specify OWNER or OWNER/REPO")
	}
	return target{Owner: repo.Owner, Repo: repo.Name}, nil
}
