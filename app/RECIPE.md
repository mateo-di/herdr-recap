# Recap

Every agent on every machine the Herdr app is connected to, in one popup,
grouped by whose move it is, each with a recap of where it got to.

## What it is

A popup mod (`mod.json`: an overlay popup at the top, 80% of the window),
opened with ⌘R. It's the Herdr app's side of
[herdr-recap](https://github.com/mrolafsson/herdr-recap), which writes the
recaps.

## What it does, and asks

- Lists `agents()` from every machine, grouped into stages: Needs you,
  Done, Needs fixing, Idle, Working, In review, Merged. Tabs for each, with
  counts; ← → move between them.
- For each machine with Claude Code agents, runs `herdr-recap sessions`
  there (through `exec(…, { on: machine })`) with the agents' Claude
  processes. It answers with each one's recap, a better title when its own
  is poor, its model, mode, last prompt, the question it's waiting on, its
  tasks, branch and git changes. Out-of-date recaps are written in the
  background by Claude Code's own `/recap`, in a fork of the session that's
  never saved.
- **Runs programs and reaches the network:** the first time it looks at a
  machine, it downloads the herdr-recap release it names (`RECAP_VERSION`
  in `main.js`) from GitHub for that machine, checks it against the
  release's `checksums.txt`, and keeps it in
  `~/.local/share/herdr-recap/app/<version>/`. It runs `hc` to read each
  agent's terminal, and `hc send` to type a reply into an agent.
- With herdr-github and herdr-linear in `~/.local/bin` where you're signed
  in, it shows each agent's pull request and Linear issue (`prs`,
  `issues`).
- Reads and writes `recap-cache.json` in the app's home, to show what it
  knew at once on opening.

## Its look

Colour only on state, from the theme's own roles (`blocked`, `working`,
`done`, `idle`, `accent`, the ANSI greens and magentas for PRs). Drawn icons
in `icons/`, 16px with a 1.6 stroke, tinted with the text colour: an arc that
spins while an agent works, a ring that breathes while it needs you. Cards
with a raised surface, a shadow and an edge in their stage's colour when
chosen; a question waiting is a callout with its own Reply; details are
chips. JetBrains Mono for branches, machines and keys.

## Keys (`keys.json`)

`cmd-r` opens it. In it: `tab` replies, `escape` leaves a reply before it
closes the popup, `ctrl-r` takes a suggested title, or asks for one now (`herdr-recap
title`, in the background), `ctrl-x` turns it down, and `ctrl-e` renames
the agent to a title you type.
`recap.demo` (with `herdr-app do`, while it's open) switches to made-up
agents for that opening.
