package flashcards

import (
	"hash/fnv"
	"sort"
	"time"
)

// CardDueInfo is scheduling status used for session selection and deck chips.
type CardDueInfo struct {
	CardID     string
	Side       string
	IsNew      bool
	IsDue      bool
	NextDueAt  *time.Time
	Learning   bool // seen in tool state but not yet due later
}

// DeckStatus summarizes learner progress for the deck header.
type DeckStatus struct {
	NewCount      int     `json:"newCount"`
	DueCount      int     `json:"dueCount"`
	LearningCount int     `json:"learningCount"`
	LaterCount    int     `json:"laterCount"`
	NextDueAt     *string `json:"nextDueAt,omitempty"`
	SRSEnabled    bool    `json:"srsEnabled"`
	TotalCards    int     `json:"totalCards"`
	RatedCards    int     `json:"ratedCards"`
}

// SelectSessionQueue picks new + due items, capped and optionally shuffled.
// dueByKey maps "cardId|side" → due info from SRS (nil means treat unseen as new).
func SelectSessionQueue(cfg Config, st State, dueByKey map[string]CardDueInfo, now time.Time, seed string) []QueueItem {
	_ = now
	items := buildCandidateItems(cfg, st, dueByKey)
	capN := cfg.SessionCap
	if capN < 1 {
		capN = 20
	}
	if len(items) > capN {
		items = items[:capN]
	}
	if cfg.Shuffle {
		shuffleDeterministic(items, seed)
	}
	return items
}

func buildCandidateItems(cfg Config, st State, dueByKey map[string]CardDueInfo) []QueueItem {
	var due, neu []QueueItem
	for _, c := range cfg.Cards {
		sides := []string{SideForward}
		if cfg.ReversePractice {
			sides = append(sides, SideReverse)
		}
		for _, side := range sides {
			key := c.ID + "|" + side
			info, ok := dueByKey[key]
			prog, hasProg := st.Cards[c.ID]
			switch {
			case ok && info.IsDue:
				due = append(due, QueueItem{CardID: c.ID, Side: side})
			case ok && info.IsNew:
				neu = append(neu, QueueItem{CardID: c.ID, Side: side})
			case !ok:
				// No SRS row: treat forward (and reverse if enabled) as new when never seen,
				// or due again when already rated in tool state (self-check mode).
				if !hasProg || prog.Seen == 0 {
					neu = append(neu, QueueItem{CardID: c.ID, Side: side})
				} else if side == SideForward {
					// Self-check / SRS-off: include previously seen forward cards as due.
					due = append(due, QueueItem{CardID: c.ID, Side: side})
				} else if side == SideReverse && (!hasProg || prog.Seen == 0) {
					neu = append(neu, QueueItem{CardID: c.ID, Side: side})
				}
			}
		}
	}
	out := make([]QueueItem, 0, len(due)+len(neu))
	out = append(out, due...)
	out = append(out, neu...)
	return out
}

// ComputeDeckStatus builds header chips from tool state + optional SRS due map.
func ComputeDeckStatus(cfg Config, st State, dueByKey map[string]CardDueInfo, srsEnabled bool) DeckStatus {
	status := DeckStatus{
		SRSEnabled: srsEnabled,
		TotalCards: len(cfg.Cards),
	}
	var earliest *time.Time
	rated := 0
	for _, c := range cfg.Cards {
		if prog, ok := st.Cards[c.ID]; ok && prog.Seen > 0 {
			rated++
		}
		sides := []string{SideForward}
		if cfg.ReversePractice {
			sides = append(sides, SideReverse)
		}
		for _, side := range sides {
			key := c.ID + "|" + side
			if info, ok := dueByKey[key]; ok {
				if info.IsNew {
					status.NewCount++
				} else if info.IsDue {
					status.DueCount++
				} else if info.NextDueAt != nil {
					status.LaterCount++
					if earliest == nil || info.NextDueAt.Before(*earliest) {
						t := *info.NextDueAt
						earliest = &t
					}
				} else {
					status.LearningCount++
				}
				continue
			}
			prog, ok := st.Cards[c.ID]
			if !ok || prog.Seen == 0 {
				status.NewCount++
			} else {
				status.LearningCount++
			}
		}
	}
	status.RatedCards = rated
	if earliest != nil {
		s := earliest.UTC().Format(time.RFC3339)
		status.NextDueAt = &s
	}
	return status
}

func shuffleDeterministic(items []QueueItem, seed string) {
	if len(items) < 2 {
		return
	}
	type keyed struct {
		item QueueItem
		h    uint64
	}
	keyedItems := make([]keyed, len(items))
	for i, it := range items {
		h := fnv.New64a()
		_, _ = h.Write([]byte(seed))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(it.CardID))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(it.Side))
		keyedItems[i] = keyed{item: it, h: h.Sum64()}
	}
	sort.SliceStable(keyedItems, func(i, j int) bool {
		return keyedItems[i].h < keyedItems[j].h
	})
	for i := range keyedItems {
		items[i] = keyedItems[i].item
	}
}
