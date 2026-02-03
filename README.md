# gh-reaction

GitHub CLI extension to see the latest emoji reactions on your project(s)

## Installation

```bash
gh extension install ccoVeille/gh-reaction
```

## Motivation

While GitHub's REST API allows you to view reactions on individual items (issues, PRs, comments), **there is no built-in endpoint to aggregate and List all reactions across a repository or user**. This creates a knowledge gap: you can't easily answer questions like:

- "What posts got the most engagement lately?"
- "Which of my recent PRs or issues resonated most with the community?"
- "Who's been most engaged with my work?"

This tool fills that gap by querying GitHub's GraphQL API to fetch all reactions across your projects and presenting them in an aggregated, human-readable format. It gives you visibility into community engagement patterns and helps maintainers understand what content resonates most with their audience.

## Usage

```console
$ gh reaction
A CLI tool to analyze GitHub reactions (emojis) on your or someone else's posts (issues, PRs, comments).

Usage:
  gh-reaction [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  repository  Analyze reactions on repository items
  user        Analyze reactions on posts made by a user

Flags:
  -h, --help   help for gh-reaction
```

Examples:

```bash
$ gh reaction user ccoVeille -since 2026-01-02
$ gh reaction repository golangci/golangci-lint -since 48h
```

