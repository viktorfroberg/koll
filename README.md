# koll

Real-time git diff viewer for your terminal.

Watches your repo for changes, shows which files changed, and displays diffs inline — updating live. Designed to run in a split pane next to your editor or AI coding agent.

Not a git client. No staging, no committing, no branching. Just diffs.

## Install

```bash
# homebrew
brew tap viktorfroberg/tap && brew install koll

# go
go install github.com/viktorfroberg/koll/cmd/koll@latest

# binary
curl -sSfL https://raw.githubusercontent.com/viktorfroberg/koll/main/install.sh | sh
```

Update with `koll --update` or `brew upgrade koll`.

## Usage

```bash
koll                  # watch current repo
koll ~/project        # watch a specific repo
koll --split          # open in a split pane (auto-detects terminal)
```

`--split` supports cmux, tmux, zellij, wezterm, kitty, ghostty, and iTerm2.

Works with worktrees — run separate instances for each:

```bash
koll ~/project-wt-auth
koll ~/project-wt-refactor
```

## Keybindings

```
j/k          jump between files
↑/↓          scroll line by line
enter/l      toggle file diff
a            expand all
c            collapse all
s            cycle filter: all → unstaged → staged
ctrl+d/u     half page scroll
g/G          top / bottom
r            force refresh
q            quit
```

## Contributing

```bash
git clone https://github.com/viktorfroberg/koll.git
cd koll
make build    # builds binary with version from git tags
make install  # copies to /usr/local/bin
```

PRs welcome. Keep it simple — koll does one thing.

## License

MIT
