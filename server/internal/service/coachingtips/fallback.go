package coachingtips

import "hash/fnv"

// FallbackTips is a curated pool used when LLM generation fails (plan 9.9).
var FallbackTips = []string{
	"Break study sessions into 25-minute focused blocks with short breaks — your brain retains more when you pause.",
	"Quiz yourself on material before re-reading notes; retrieval practice beats passive review.",
	"Schedule your hardest topics at the time of day you study most consistently.",
	"Explain a concept aloud as if teaching a friend — gaps in your explanation show what to review next.",
	"Sleep consolidates memory; avoid all-nighters before assessments when you can plan ahead.",
	"Switch topics every hour to stay fresh, but finish one practice problem set before moving on.",
	"Write one sentence summarizing each section before moving to the next — it sharpens focus.",
	"Use practice tests under light time pressure to mirror exam conditions.",
	"Review missed quiz questions within 48 hours while the reasoning is still fresh.",
	"Keep a running list of terms you confuse; drill those pairs specifically.",
	"Study in the same place when possible — context cues help recall.",
	"Start each session by recalling what you learned last time without looking at notes.",
	"Pair reading with a quick sketch or diagram to engage visual memory.",
	"Set a concrete goal for each session (e.g., finish two problem sets) instead of vague time targets.",
	"When stuck, write what you do understand before asking for help — it clarifies the gap.",
	"Alternate between courses so one subject does not crowd out the rest of your week.",
	"Turn headings into questions and answer them from memory after reading.",
	"Reduce distractions by leaving your phone in another room for focused blocks.",
	"Celebrate small wins — finishing a module counts toward long-term mastery.",
	"If scores plateau despite long hours, change method: try practice tests or teach-back instead of more reading.",
}

// PickFallback returns a deterministic tip from the pool for a user/week seed.
func PickFallback(userSeed string) string {
	if len(FallbackTips) == 0 {
		return "Keep showing up — consistent short sessions beat occasional marathons."
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(userSeed))
	return FallbackTips[int(h.Sum32())%len(FallbackTips)]
}
