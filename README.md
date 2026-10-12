# prv

prv is a terminal UI that shows the state of your GitHub pull requests.
It replaces a pile of browser tabs that you refresh by hand.

You can see, without opening a browser, which of your PRs are approved,
blocked or failing CI.

prv lists the PRs you authored or are assigned to. The list has every
open PR, plus the PRs merged or closed in the last 14 days.
Each row shows the state, the review status and the CI status.

## Install

You need Go 1.26 or newer.

```sh
go install github.com/sbartram/prv/cmd/prv@latest
```

Make sure your Go bin directory is on your `PATH`. Then check the install:

```sh
prv --version
```

The command prints one line, for example `prv v0.1.0`.

## Authentication

prv uses the same token as the `gh` command. It looks for a token in this order:

1. `GH_TOKEN`
2. `GITHUB_TOKEN`
3. The `gh` login (run `gh auth login` to create it)

A token in the environment overrides the `gh` login.
If that token is old or revoked, prv shows a 401 error.
Fix the token or unset the variable.

prv never stores, logs or displays your token.

If prv finds no token, it prints a message and exits with status 1.

### Least privilege

prv only reads data. It never asks for write access.

- The default scopes of the `gh` login work.
- A fine-grained token needs read-only access to these permissions on the
  repositories you watch: Pull requests, Checks, Commit statuses and Metadata.

## Keys

| Key              | Action                         |
| ---------------- | ------------------------------ |
| `j` or `down`    | Move the selection down        |
| `k` or `up`      | Move the selection up          |
| `r`              | Retry (from the error screen)  |
| `q` or `ctrl+c`  | Quit                           |

prv fetches the data once, when it starts.

## Legend

### STATE

| Text     | Meaning                                         |
| -------- | ----------------------------------------------- |
| `open`   | The PR is open                                  |
| `merged` | The PR is merged                                |
| `closed` | The PR is closed and not merged                 |

A draft PR shows `[draft]` before its title. Its row is dimmed.

### REVIEW

| Badge        | Meaning                                  |
| ------------ | ---------------------------------------- |
| `✓ approved` | The PR is approved                       |
| `✗ changes`  | A reviewer asked for changes             |
| `○ required` | A review is required and not done yet    |
| `– none`     | The PR has no review status              |

### CI

| Badge         | Meaning                                |
| ------------- | -------------------------------------- |
| `✓ passing`   | All checks passed                      |
| `✗ failing`   | At least one check failed              |
| `● running`   | Checks are still running               |
| `– no checks` | The PR has no checks                   |
| `? unknown`   | prv cannot tell the CI status          |

## License

See [LICENSE](LICENSE).

<!-- uat scratch: 2026-10-12T01:36:02Z -->
