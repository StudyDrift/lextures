import { describe, expect, it } from 'vitest'
import type { FeatureHelpTopic } from '../../context/feature-help-context'
import { FEATURE_HELP_BODY, FEATURE_HELP_MEDIA, FEATURE_HELP_TITLES } from '../feature-help-content'

const TOPICS: FeatureHelpTopic[] = [
  'gradebook',
  'modules',
  'question-bank',
  'quiz-authoring',
  'syllabus',
  'content-page',
]

describe('feature-help content registry (plan W06)', () => {
  it('defines titles and body copy for every topic', () => {
    for (const topic of TOPICS) {
      expect(FEATURE_HELP_TITLES[topic].length).toBeGreaterThan(0)
      expect(FEATURE_HELP_BODY[topic].length).toBeGreaterThan(0)
    }
  })

  it('does not ship placeholder copy in help body text', () => {
    for (const topic of TOPICS) {
      expect(FEATURE_HELP_BODY[topic]).not.toMatch(/placeholder/i)
      expect(FEATURE_HELP_BODY[topic]).not.toMatch(/when ready/i)
    }
  })

  it('wires optional media with src and alt keys only', () => {
    for (const [topic, media] of Object.entries(FEATURE_HELP_MEDIA)) {
      expect(TOPICS).toContain(topic)
      expect(media?.src.startsWith('/')).toBe(true)
      expect(media?.altKey.startsWith('featureHelp.media.')).toBe(true)
    }
  })
})