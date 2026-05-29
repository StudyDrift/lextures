import { useEffect, useState } from 'react'
import {
  accommodationSpeechToTextEnabled,
  fetchMyAccommodations,
  fetchReadingPreferences,
} from '../lib/reading-preferences-api'
import { speechToTextFeatureEnabled } from '../lib/platform-features'

export type SpeechToTextAvailability = {
  enabled: boolean
  language: string
  accommodationTooltip?: string
  loading: boolean
}

export function useSpeechToTextAvailability(courseCode?: string): SpeechToTextAvailability {
  const [language, setLanguage] = useState('en-US')
  const [accommodation, setAccommodation] = useState(false)
  const [loading, setLoading] = useState(true)

  const featureOn = speechToTextFeatureEnabled()

  useEffect(() => {
    if (!featureOn) {
      setLoading(false)
      return
    }
    let cancelled = false
    void (async () => {
      try {
        const [prefs, accommodations] = await Promise.all([
          fetchReadingPreferences().catch(() => ({ sttEnabled: false, sttLanguage: 'en-US' })),
          fetchMyAccommodations(),
        ])
        if (cancelled) return
        setLanguage(prefs.sttLanguage || 'en-US')
        setAccommodation(accommodationSpeechToTextEnabled(accommodations, courseCode))
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [featureOn, courseCode])

  const enabled = featureOn
  const accommodationTooltip = accommodation
    ? 'Speech-to-text enabled by your accommodation plan.'
    : undefined

  return { enabled, language, accommodationTooltip, loading }
}
