export function insightLabelKey(insightKey: string): string {
  const map: Record<string, string> = {
    peak_study_window: 'learnerProfile.insight.label.peakStudyWindow',
    study_consistency: 'learnerProfile.insight.label.studyConsistency',
    study_streak: 'learnerProfile.insight.label.studyStreak',
    session_shape: 'learnerProfile.insight.label.sessionShape',
    modality_affinity: 'learnerProfile.insight.label.modalityAffinity',
    complexity_comfort: 'learnerProfile.insight.label.complexityComfort',
    content_pacing: 'learnerProfile.insight.label.contentPacing',
    top_strengths: 'learnerProfile.insight.label.topStrengths',
    growth_areas: 'learnerProfile.insight.label.growthAreas',
    needs_review: 'learnerProfile.insight.label.needsReview',
    persistence: 'learnerProfile.insight.label.persistence',
    help_seeking: 'learnerProfile.insight.label.helpSeeking',
    consolidation: 'learnerProfile.insight.label.consolidation',
  }
  if (insightKey.startsWith('topic_')) {
    return 'learnerProfile.insight.label.topic'
  }
  return map[insightKey] ?? 'learnerProfile.insight.genericUnknown'
}