# koll

Real-time git diff viewer for your terminal. Swedish for "check" — as in *hålla koll* (keep an eye on).

You moved your dev workflow to the terminal. Claude Code, Codex, or whatever agent is doing the typing now. It's fast, it's great — but you miss one thing from your IDE: seeing what's actually changing.

Lazygit? Overkill. You don't need a full git client. You just want to *see the diff* — which files changed, what got added, what got deleted — updating live as your agent works.

That's koll. One pane. One job. Keep an eye on things.

```
┌──────────────────────────────────────┐
│  koll  ~/project                     │
├──────────────────────────────────────┤
│  M src/stores/booking.js             │
│ >A tests/booking.spec.js            │
│  │ +import { describe } from 'vi..  │
│  │ +describe('Booking', () => {     │
│  │ +  it('loads schedule', () => {  │
│  │ -  // old implementation         │
│  D src/old-booking.js                │
├──────────────────────────────────────┤
│  3 files · +12 -3   q:uit a:ll      │
└──────────────────────────────────────┘
```

## Install

```bash
# homebrew
brew tap viktorfroberg/tap && brew install koll

# go
go install github.com/viktorfroberg/koll/cmd/koll@latest

# binary
curl -sSfL https://raw.githubusercontent.com/viktorfroberg/koll/main/install.sh | sh
```

## Update

```bash
koll --update
```

Works regardless of how you installed it (curl, go install, or manual). Checks GitHub for the latest release and replaces the binary in place. Homebrew users can just `brew upgrade koll`.

## Usage

```bash
koll                  # watch current repo
koll ~/project        # watch a specific repo
koll --split          # open in a split pane next to your agent
```

### With Claude Code

```
! koll --split
```

That's it. A split pane opens to the right with your live diff view.

### With worktrees

Running multiple agents in parallel? Point each koll at its own worktree:

```bash
koll ~/project-wt-auth      # pane 1
koll ~/project-wt-refactor   # pane 2
```

### Split pane support

`--split` auto-detects your terminal and opens koll in an adjacent pane:

cmux, tmux, zellij, wezterm, kitty, ghostty, iTerm2

## Keybindings

```
j/k  ↑/↓       navigate files
enter  l        toggle diff
a               expand all
c               collapse all
s               cycle: all → unstaged → staged
r               force refresh
q               quit
```

## How it works

koll watches your repo for filesystem changes, debounces rapid writes (300ms), and re-runs `git diff`. Diffs are lazy-loaded — only fetched when you expand a file. The whole thing is a single 4MB Go binary with zero runtime dependencies.

## License

MIT
