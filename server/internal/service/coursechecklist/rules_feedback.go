package coursechecklist

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func feedbackRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleFeedbackRubricsOnHighStakes(),
		ruleFeedbackCriteriaPublished(),
		ruleFeedbackFormativePerModule(),
		ruleFeedbackQuizReviewSettings(),
		ruleFeedbackAttemptsPolicy(),
		ruleFeedbackPeerReviewConfig(),
	}
}

func ruleFeedbackRubricsOnHighStakes() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemFeedbackRubricsOnHighStakes,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.feedback.rubrics-on-high-stakes.title",
		TitleDefault: "Add criteria to high-stakes work",
		WhyKey:       "coursechecklist.item.feedback.rubrics-on-high-stakes.why",
		WhyDefault:   "High-stakes work needs published criteria so grading is defensible.",
		HelpRef:      "course-checklist#feedback-rubrics-on-high-stakes",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.3", "OSCQR 46"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems, DataNeedGrading, DataNeedStructure},
		Evaluate:     evalFeedbackRubricsOnHighStakes,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "assignment.rubric",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
		Action: &ItemAction{
			Kind:       ActionKindBuildRubricAI,
			LabelKey:   "coursechecklist.action.build_rubric_ai",
			Label:      "Build a rubric with AI",
			Endpoint:   "/api/v1/courses/{courseCode}/assignments/{itemId}/generate-rubric",
			RequiresAI: true,
		},
	}
}

func evalFeedbackRubricsOnHighStakes(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := sortAssessmentItems(assessmentItemsFor(snap))
	total := totalCoursePoints(items)
	if total <= 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.feedback.rubrics-on-high-stakes.detail.na",
			DetailDefault: "Does not apply when the course has no points.",
		}, nil
	}
	weighted := usesWeightedGrading(snap)
	var evidence []EvidenceRow
	for _, it := range items {
		if !isHighStakes(it, total, weighted) {
			continue
		}
		if it.HasRubric || it.HasBody {
			continue
		}
		pct := 100.0 * float64(*it.Points) / float64(total)
		evidence = append(evidence, assessmentEvidenceRow(it,
			fmt.Sprintf("%.0f%% of course with no criteria", pct), "assignment.rubric"))
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.feedback.rubrics-on-high-stakes.detail.done",
			DetailDefault: "High-stakes work has rubrics or written criteria.",
			DetailFields:  map[string]any{"totalPoints": total},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.feedback.rubrics-on-high-stakes.detail.todo",
		DetailDefault: fmt.Sprintf("%d high-stakes items need rubrics or written criteria.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence), "totalPoints": total},
		Evidence:      evidence,
	}, nil
}

func ruleFeedbackCriteriaPublished() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemFeedbackCriteriaPublished,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.feedback.criteria-published.title",
		TitleDefault: "Write instructions for every assessment",
		WhyKey:       "coursechecklist.item.feedback.criteria-published.why",
		WhyDefault:   "Learners need instructions before they can complete the work.",
		HelpRef:      "course-checklist#feedback-criteria-published",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.3", "OSCQR 46"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems},
		Evaluate:     evalFeedbackCriteriaPublished,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalFeedbackCriteriaPublished(_ context.Context, snap CourseSnapshot) (Finding, error) {
	items := sortAssessmentItems(assessmentItemsFor(snap))
	if len(items) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.feedback.criteria-published.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	var evidence []EvidenceRow
	for _, it := range items {
		if !it.HasBody {
			evidence = append(evidence, assessmentEvidenceRow(it, "No instructions", ""))
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.feedback.criteria-published.detail.done",
			DetailDefault: "Every assessment has instructions.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.feedback.criteria-published.detail.todo",
		DetailDefault: fmt.Sprintf("%d assessments have no instructions.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleFeedbackFormativePerModule() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemFeedbackFormativePerModule,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.feedback.formative-per-module.title",
		TitleDefault: "Add a low-stakes check to each module",
		WhyKey:       "coursechecklist.item.feedback.formative-per-module.why",
		WhyDefault:   "Formative checks keep courses from being all high-stakes finals.",
		HelpRef:      "course-checklist#feedback-formative-per-module",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.4", "OSCQR 47", "NSQ D"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedAssessmentItems, DataNeedItemMeta, DataNeedContentTools},
		Evaluate:     evalFeedbackFormativePerModule,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module", "Issue"}},
	}
}

func evalFeedbackFormativePerModule(_ context.Context, snap CourseSnapshot) (Finding, error) {
	mods := listModules(snap)
	if len(mods) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.feedback.formative-per-module.detail.na",
			DetailDefault: "Does not apply until the course has modules.",
		}, nil
	}
	items := assessmentItemsFor(snap)
	total := totalCoursePoints(items)
	children := childrenByParent(snap)
	var evidence []EvidenceRow
	for _, mod := range mods {
		if moduleHasFormative(snap, mod, children[mod.ID], items, total) {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:    mod.Title,
			Sublabel: "No low-stakes formative check",
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.feedback.formative-per-module.detail.done",
			DetailDefault: "Every module has a low-stakes formative check.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.feedback.formative-per-module.detail.todo",
		DetailDefault: fmt.Sprintf("%d modules need a formative check.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func moduleHasFormative(snap CourseSnapshot, mod StructureItem, kids []StructureItem, assessments []AssessmentItemSnap, total int) bool {
	byID := map[string]AssessmentItemSnap{}
	for _, a := range assessments {
		byID[a.ID.String()] = a
	}
	for _, kid := range kids {
		if kid.Archived {
			continue
		}
		// Survey / practice quiz / content tool / SRS-ish interactive.
		if kid.Kind == "survey" {
			return true
		}
		if snap.ContentToolItemIDs != nil {
			if _, ok := snap.ContentToolItemIDs[kid.ID]; ok {
				return true
			}
		}
		a, ok := byID[kid.ID.String()]
		if !ok {
			// Ungraded practice: assignment/quiz with 0 points via ItemMeta.
			if meta, has := snap.ItemMeta[kid.ID]; has && isGradableKind(kid.Kind) {
				if meta.PointsWorth != nil && *meta.PointsWorth <= 0 {
					return true
				}
			}
			continue
		}
		if a.Points == nil || *a.Points <= 0 {
			return true
		}
		if total > 0 {
			pct := 100.0 * float64(*a.Points) / float64(total)
			if pct < 2.0 {
				return true
			}
		}
	}
	_ = mod
	return false
}

func ruleFeedbackQuizReviewSettings() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemFeedbackQuizReviewSettings,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.feedback.quiz-review-settings.title",
		TitleDefault: "Review what quizzes reveal, and when",
		WhyKey:       "coursechecklist.item.feedback.quiz-review-settings.why",
		WhyDefault:   "Revealing correct answers while others can still take the quiz is an integrity defect.",
		HelpRef:      "course-checklist#feedback-quiz-review-settings",
		Tier:         TierRecommended,
		Sources:      []string{"QM 3.5", "OSCQR 45"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems},
		Evaluate:     evalFeedbackQuizReviewSettings,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "quiz.scores-review",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalFeedbackQuizReviewSettings(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var quizzes []AssessmentItemSnap
	for _, it := range assessmentItemsFor(snap) {
		if it.Kind == "quiz" {
			quizzes = append(quizzes, it)
		}
	}
	if len(quizzes) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.feedback.quiz-review-settings.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	now := time.Now().UTC()
	var evidence []EvidenceRow
	for _, q := range sortAssessmentItems(quizzes) {
		// Graded quiz revealing correct answers immediately while availability still open.
		graded := q.Points != nil && *q.Points > 0
		revealsCorrect := strings.EqualFold(q.ReviewVisibility, "correct_answers") ||
			strings.EqualFold(q.ReviewVisibility, "full")
		immediate := strings.EqualFold(q.ReviewWhen, "after_submit") ||
			strings.EqualFold(q.ShowScoreTiming, "immediate")
		windowOpen := q.AvailableUntil == nil || q.AvailableUntil.After(now)
		if graded && revealsCorrect && immediate && windowOpen {
			evidence = append(evidence, assessmentEvidenceRow(q,
				"Reveals correct answers while still available", "quiz.scores-review"))
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.feedback.quiz-review-settings.detail.done",
			DetailDefault: "Quiz review settings look consistent.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.feedback.quiz-review-settings.detail.todo",
		DetailDefault: fmt.Sprintf("%d quizzes reveal answers while still available.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleFeedbackAttemptsPolicy() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemFeedbackAttemptsPolicy,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.feedback.attempts-policy.title",
		TitleDefault: "Make attempts and scoring consistent",
		WhyKey:       "coursechecklist.item.feedback.attempts-policy.why",
		WhyDefault:   "One attempt cannot use highest-of-attempts scoring.",
		HelpRef:      "course-checklist#feedback-attempts-policy",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 45"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems},
		Evaluate:     evalFeedbackAttemptsPolicy,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
			Anchor:  "quiz.attempts-grading",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalFeedbackAttemptsPolicy(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var evidence []EvidenceRow
	var quizCount int
	for _, q := range sortAssessmentItems(assessmentItemsFor(snap)) {
		if q.Kind != "quiz" {
			continue
		}
		quizCount++
		if q.UnlimitedAttempts {
			continue
		}
		if q.MaxAttempts <= 1 {
			pol := strings.ToLower(strings.TrimSpace(q.GradeAttemptPolicy))
			if pol == "highest" || pol == "average" {
				evidence = append(evidence, assessmentEvidenceRow(q,
					fmt.Sprintf("1 attempt with %s-of-attempts scoring", pol), "quiz.attempts-grading"))
			}
		}
	}
	if quizCount == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.feedback.attempts-policy.detail.na",
			DetailDefault: "Does not apply to this course.",
		}, nil
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.feedback.attempts-policy.detail.done",
			DetailDefault: "Quiz attempts and scoring policies are consistent.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.feedback.attempts-policy.detail.todo",
		DetailDefault: fmt.Sprintf("%d quizzes have inconsistent attempt policies.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}

func ruleFeedbackPeerReviewConfig() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemFeedbackPeerReviewConfig,
		Category:     CategoryFeedback,
		TitleKey:     "coursechecklist.item.feedback.peer-review-config.title",
		TitleDefault: "Complete your peer-review setup",
		WhyKey:       "coursechecklist.item.feedback.peer-review-config.why",
		WhyDefault:   "Peer review needs an allocation strategy, a review window after the due date, and a rubric.",
		HelpRef:      "course-checklist#feedback-peer-review-config",
		Tier:         TierRecommended,
		Sources:      []string{"QM 5.2"},
		DataNeeds:    []DataNeed{DataNeedPeerReview},
		Applies: func(snap CourseSnapshot) bool {
			return len(snap.PeerReviewConfigs) > 0
		},
		Evaluate: evalFeedbackPeerReviewConfig,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: assessmentEvidenceColumns},
	}
}

func evalFeedbackPeerReviewConfig(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var evidence []EvidenceRow
	for _, p := range snap.PeerReviewConfigs {
		var issues []string
		if p.ReviewsPerReviewer < 1 {
			issues = append(issues, "no allocation strategy")
		}
		if p.OpensAt == nil || (p.DueAt != nil && p.OpensAt.Before(*p.DueAt)) {
			issues = append(issues, "review window should open after the due date")
		}
		if !p.HasRubric {
			issues = append(issues, "missing rubric")
		}
		if len(issues) == 0 {
			continue
		}
		route := "/courses/{courseCode}/modules/assignment/{itemId}"
		route = strings.ReplaceAll(route, "{itemId}", p.AssignmentID.String())
		evidence = append(evidence, EvidenceRow{
			Label:    p.AssignmentTitle,
			Sublabel: strings.Join(issues, "; "),
			TargetOverride: &NavTarget{
				Surface:   "web",
				Route:     route,
				EntityKey: p.AssignmentID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.feedback.peer-review-config.detail.done",
			DetailDefault: "Peer-review configurations look complete.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.feedback.peer-review-config.detail.todo",
		DetailDefault: fmt.Sprintf("%d peer-reviewed assignments need setup.", len(evidence)),
		DetailFields:  map[string]any{"count": len(evidence)},
		Evidence:      evidence,
	}, nil
}
