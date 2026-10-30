package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

// What the Herdr app's mod (mods-recap) asks of this machine. The app runs
// its agents in herdr-core, not the herdr server, so it names an agent by its
// Claude process rather than a herdr pane, and runs this where the agent runs.

// sessionAsk is one agent the mod shows: its Claude process and its title.
type sessionAsk struct {
	PID   int    `json:"pid"`
	Title string `json:"title,omitempty"`
}

// sessionInfo is what the mod is told of it. Recap is the cached one, written
// for an earlier point when Fresh is false; Error says why there's none.
type sessionInfo struct {
	PID     int       `json:"pid"`
	Session string    `json:"session,omitempty"`
	Recap   string    `json:"recap,omitempty"`
	At      time.Time `json:"at,omitzero"`
	Fresh   bool      `json:"fresh"`
	Title   string    `json:"title,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// runSessions prints each asked-for agent's session and cached recap as JSON.
// --write: recaps out of date are written too, by a copy of this program that
// carries on after this one returns (the mod's calls are cut off at 30 s, and
// a recap can take longer); the mod finds them on its next ask. --now (what
// that copy runs): write them here and now.
func runSessions(ctx context.Context, cfg config, args []string) error {
	fs := flag.NewFlagSet("sessions", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	write := fs.Bool("write", false, "")
	now := fs.Bool("now", false, "")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("sessions: %w", err)
	}
	if fs.NArg() != 1 {
		return errors.New(`sessions needs one argument: [{"pid": N, "title": "…"}, …]`)
	}
	var asks []sessionAsk
	if err := json.Unmarshal([]byte(fs.Arg(0)), &asks); err != nil {
		return fmt.Errorf("sessions: %w", err)
	}

	if *now {
		for _, a := range asks {
			s, err := sessionForPID(a.PID, "claude")
			if err == nil {
				_, _, err = ensureRecap(ctx, cfg, s, a.Title, false)
			}
			if err != nil && !errors.Is(err, errNothingYet) {
				fmt.Fprintln(os.Stderr, "herdr-recap: recap of", a.PID, "failed:", err)
			}
		}
		return nil
	}

	out := make([]sessionInfo, 0, len(asks))
	var stale []sessionAsk
	for _, a := range asks {
		info := sessionInfo{PID: a.PID}
		s, err := sessionForPID(a.PID, "claude")
		if err != nil {
			info.Error = err.Error()
			out = append(out, info)
			continue
		}
		info.Session = s.ID
		c := loadRecap(s.ID)
		info.Fresh = fresh(c, s)
		if c != nil {
			info.Recap, info.At, info.Title = c.Text, c.At, c.Title
		}
		if !info.Fresh && s.Transcript != "" {
			stale = append(stale, a)
		}
		out = append(out, info)
	}
	if *write && len(stale) > 0 {
		ask, _ := json.Marshal(stale)
		if err := spawn("sessions", "--now", string(ask)); err != nil {
			fmt.Fprintln(os.Stderr, "herdr-recap: writing recaps:", err)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
