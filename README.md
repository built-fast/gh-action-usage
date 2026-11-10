# gh-action-usage

A GitHub CLI extension that shows GitHub Actions billing usage for an
organization, user, or repository.

## Installation

```sh
gh extension install built-fast/gh-action-usage
```

## Usage

```sh
gh action-usage                       # repository in the current directory
gh action-usage built-fast            # whole organization (or user)
gh action-usage built-fast/foo        # single repository
gh action-usage -m 2026-10 built-fast # a specific month (default: current)
gh action-usage --by-repo built-fast  # per-repository breakdown for an owner
gh action-usage --by-repo             # same, for the current repo's owner
```

Organization billing requires the `admin:org` scope:

```sh
gh auth refresh -s admin:org
```

## Development

```sh
make test    # vet, format check, tests
make build   # build ./gh-action-usage
make install # install the local build as a gh extension
```

## License

MIT License. See [LICENSE](LICENSE) for details.
