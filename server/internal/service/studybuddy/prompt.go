package studybuddy

import (
	"fmt"
	"strings"

	studybuddyrepo "github.com/lextures/lextures/server/internal/repos/studybuddy"
)

const systemPromptTemplate = `You are an AI study buddy for the course "{COURSE_TITLE}". You help self-learners understand course material, review concepts, and stay on track with their goals.

Learner context:
- Display name: {DISPLAY_NAME}
- Prior knowledge level: {PRIOR_LEVEL}
{GOALS_BLOCK}{STRUGGLES_BLOCK}{SESSION_BLOCK}

Rules:
- You are an AI assistant. Be warm, encouraging, and concise. Use Markdown when helpful.
- {GROUNDING_RULE}
- Always cite the specific module item name when referencing course material.
- Personalize examples to the learner's stated goals when relevant.
- Do not write assignments or graded work for the learner; explain concepts and guide practice instead.
- If the learner asks about topics outside their enrolled course materials, say clearly: "I couldn't find information about this in your course materials."
- Include a brief reminder that you are AI when answering the first question in a session.`

func BuildSystemPrompt(
	courseTitle string,
	memory *studybuddyrepo.MemoryRow,
	priorLevel, displayName string,
	hasRAG bool,
) string {
	if strings.TrimSpace(priorLevel) == "" {
		priorLevel = "beginner"
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = "Learner"
	}
	grounding := `Ground every factual claim in the provided course excerpts. If the excerpts do not contain enough information, say you could not find it in their course materials rather than guessing.`
	if !hasRAG {
		grounding = `No course excerpts were retrieved for this question. Do not invent course-specific facts; respond with general guidance and label it as general knowledge, not from course materials.`
	}
	goalsBlock := ""
	if memory != nil && memory.GoalsSummary != nil && strings.TrimSpace(*memory.GoalsSummary) != "" {
		goalsBlock = fmt.Sprintf("- Learning goals: %s\n", strings.TrimSpace(*memory.GoalsSummary))
	}
	strugglesBlock := ""
	if memory != nil && len(memory.StruggleConcepts) > 0 {
		strugglesBlock = fmt.Sprintf("- Recent struggle areas: %s\n", strings.Join(memory.StruggleConcepts, ", "))
	}
	sessionBlock := ""
	if memory != nil && memory.LastSessionSummary != nil && strings.TrimSpace(*memory.LastSessionSummary) != "" {
		sessionBlock = fmt.Sprintf("- Last session summary: %s\n", strings.TrimSpace(*memory.LastSessionSummary))
	}
	out := systemPromptTemplate
	out = strings.ReplaceAll(out, "{COURSE_TITLE}", courseTitle)
	out = strings.ReplaceAll(out, "{DISPLAY_NAME}", displayName)
	out = strings.ReplaceAll(out, "{PRIOR_LEVEL}", priorLevel)
	out = strings.ReplaceAll(out, "{GOALS_BLOCK}", goalsBlock)
	out = strings.ReplaceAll(out, "{STRUGGLES_BLOCK}", strugglesBlock)
	out = strings.ReplaceAll(out, "{SESSION_BLOCK}", sessionBlock)
	out = strings.ReplaceAll(out, "{GROUNDING_RULE}", grounding)
	return out
}
