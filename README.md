# gh-reaction

GitHub CLI extension to see the latest emoji reactions on your project(s)

## Installation

```bash
gh extension install ccoVeille/gh-reaction
```

## Usage

```console
$ gh reaction --help

Available flags:
  -author string
        Limit to messages authored by this GitHub username
  -limit int
        Maximum number of messages to fetch (default 50)
  -since value
        Fetch messages since this date (e.g., "2023-01-02", "2h", "15m", "3d" ...) (default "90d")
```

Example:

```bash
$ gh reaction
$ gh reaction -author ccoVeille -limit 100
$ gh reaction -since 2023-01-02 -limit 0
```

You can also use

```bash
GH_REPO=owner/repo gh reaction
```