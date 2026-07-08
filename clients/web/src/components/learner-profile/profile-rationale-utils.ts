export function profileRationaleFacetPath(facetKey: string): string {
  const map: Record<string, string> = {
    study_rhythm: 'study-rhythm',
    content_modality: 'content-modality',
    strengths_growth: 'strengths-growth',
    interests: 'interests',
    learning_approach: 'learning-approach',
  }
  return map[facetKey] ?? facetKey
}