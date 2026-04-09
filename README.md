# Pier

A TUI-based SSH connection manager built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Reads your `~/.ssh/config`, organizes hosts by group, and lets you connect with a single keystroke.

## Install

```bash
go install github.com/miyago9267/ssh-pier/cmd/pier@latest
```

Or download a binary from [Releases](https://github.com/miyago9267/ssh-pier/releases).

## Usage

```bash
pier
```

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Connect to host / Toggle group |
| `Space` | Toggle group |
| `/` | Search (fuzzy match on alias, hostname, user) |
| `e` | Edit selected host |
| `n` | New host |
| `d` | Delete host (with confirmation) |
| `q` | Quit |

## Grouping

Add `# @group:` annotations in your `~/.ssh/config`:

```ssh-config
# @group: company
Host prod
    Hostname 10.0.1.100
    User deploy

Host staging
    Hostname 10.0.1.200
    User deploy

# @group: personal
Host my-vps
    Hostname 203.0.113.50
    User miyago
```

Hosts inherit the most recent `# @group:` annotation. Unannotated hosts at the top of the file are grouped as `ungrouped`.

## How It Works

- Parses `~/.ssh/config` including standard directives (Host, Hostname, User, Port, IdentityFile)
- Edits write back to `~/.ssh/config` with automatic `.bak` backup
- Connects via `exec ssh <alias>`, so your full SSH config (keys, agent, ProxyJump, etc.) is respected

## License

[MIT](LICENSE)
