export type EditorialPillar = {
  id: 'p1' | 'p2' | 'p3' | 'p4' | 'p5' | 'p6'
  slug: string
  title: string
  audience: string
  targetArticles: number
  productHref: string
}

export const EDITORIAL_PILLARS: EditorialPillar[] = [
  { id: 'p1', slug: 'adaptive-learning', title: 'Adaptive learning: how it actually works', audience: 'Higher education · K–12', targetArticles: 18, productHref: '/platform/adaptive-learning' },
  { id: 'p2', slug: 'assessment-design-ai', title: 'Assessment design in the age of generative AI', audience: 'Higher education · K–12', targetArticles: 20, productHref: '/platform/assessment' },
  { id: 'p3', slug: 'grading-and-integrity', title: 'Grading, feedback and academic integrity', audience: 'Higher education · K–12', targetArticles: 16, productHref: '/platform/grading' },
  { id: 'p4', slug: 'mastery-and-standards', title: 'Standards, outcomes and mastery-based grading', audience: 'K–12', targetArticles: 14, productHref: '/k12' },
  { id: 'p5', slug: 'choosing-an-lms', title: 'Choosing and running a learning platform', audience: 'K–12 · Higher education', targetArticles: 16, productHref: '/platform' },
  { id: 'p6', slug: 'homeschool-teaching', title: 'Teaching at home: curriculum, pacing and evidence', audience: 'Homeschool', targetArticles: 12, productHref: '/homeschool' },
]

export function editorialPillar(id: string) {
  return EDITORIAL_PILLARS.find(pillar => pillar.id === id)
}

export function pillarHref(id: string) {
  const pillar = editorialPillar(id)
  return pillar ? `/resources/guides#${pillar.id}` : '/resources/guides'
}
