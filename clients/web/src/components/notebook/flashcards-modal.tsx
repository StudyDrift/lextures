import { useCallback, useEffect, useState } from 'react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { AiDisclosureBanner } from '../ai-disclosure-banner'
import {
  X,
  Sparkles,
  ArrowLeft,
  ArrowRight,
  RefreshCw,
  CheckCircle2,
  HelpCircle,
  RotateCw,
  Award
} from 'lucide-react'

type Flashcard = {
  front: string
  back: string
}

type FlashcardsModalProps = {
  open: boolean
  notes: string
  pageTitle: string
  onClose: () => void
}

const STUDY_TIPS = [
  'Active retrieval practice is one of the most effective ways to build long-term memory.',
  'Try to formulate the answer in your head or speak it aloud before flipping the card.',
  'Spaced repetition works best when you study cards multiple times over several days.',
  'If a card is difficult, mark it as "Needs Practice" to review it again at the end.',
]

export function FlashcardsModal({ open, notes, pageTitle, onClose }: FlashcardsModalProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [flashcards, setFlashcards] = useState<Flashcard[]>([])
  const [currentIndex, setCurrentIndex] = useState(0)
  const [isFlipped, setIsFlipped] = useState(false)
  const [learned, setLearned] = useState<Record<number, boolean>>({})
  const [tipIndex, setTipIndex] = useState(0)

  // Fetch flashcards from the AI endpoint
  const generateFlashcards = useCallback(async () => {
    if (!notes.trim()) {
      setError('Please write some notes first before generating flashcards.')
      return
    }
    setLoading(true)
    setError(null)
    setFlashcards([])
    setCurrentIndex(0)
    setIsFlipped(false)
    setLearned({})

    // Cycle through study tips while loading
    const tipInterval = setInterval(() => {
      setTipIndex((prev) => (prev + 1) % STUDY_TIPS.length)
    }, 4000)

    try {
      const res = await authorizedFetch('/api/v1/me/notebooks/flashcards', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notes }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        setError(readApiErrorMessage(raw))
        return
      }
      const data = raw as { flashcards?: Flashcard[] }
      if (!data.flashcards || data.flashcards.length === 0) {
        setError('No flashcards could be generated from these notes. Try expanding your notes.')
        return
      }
      setFlashcards(data.flashcards)
    } catch {
      setError('Could not connect to the server to generate flashcards.')
    } finally {
      clearInterval(tipInterval)
      setLoading(false)
    }
  }, [notes])

  // Trigger generation on open
  useEffect(() => {
    if (open) {
      void generateFlashcards()
    }
  }, [open, generateFlashcards])

  // Keyboard navigation
  useEffect(() => {
    if (!open || loading || flashcards.length === 0) return

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === ' ') {
        e.preventDefault()
        setIsFlipped((prev) => !prev)
      } else if (e.key === 'ArrowRight') {
        e.preventDefault()
        handleNext()
      } else if (e.key === 'ArrowLeft') {
        e.preventDefault()
        handlePrev()
      } else if (e.key === 'Enter') {
        e.preventDefault()
        handleToggleLearned()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, loading, flashcards.length, currentIndex, isFlipped])

  const handleNext = () => {
    if (currentIndex < flashcards.length - 1) {
      setIsFlipped(false)
      // Allow flip animation back before changing text
      setTimeout(() => {
        setCurrentIndex((prev) => prev + 1)
      }, 150)
    }
  };

  const handlePrev = () => {
    if (currentIndex > 0) {
      setIsFlipped(false)
      setTimeout(() => {
        setCurrentIndex((prev) => prev - 1)
      }, 150)
    }
  };

  const handleToggleLearned = () => {
    setLearned((prev) => {
      const next = { ...prev, [currentIndex]: !prev[currentIndex] }
      // Auto advance on learned if not on the last card
      if (!prev[currentIndex] && currentIndex < flashcards.length - 1) {
        setTimeout(() => {
          handleNext()
        }, 300)
      }
      return next
    })
  }

  const handleResetStudy = () => {
    setCurrentIndex(0)
    setIsFlipped(false)
    setLearned({})
  }

  if (!open) return null

  const totalCards = flashcards.length
  const learnedCount = Object.values(learned).filter(Boolean).length
  const percentComplete = totalCards > 0 ? Math.round((learnedCount / totalCards) * 100) : 0
  const isMastered = totalCards > 0 && learnedCount === totalCards

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6" role="dialog" aria-modal="true">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-slate-900/60 backdrop-blur-sm transition-opacity" 
        onClick={onClose}
      />

      {/* Modal Card */}
      <div className="relative flex flex-col w-full max-w-2xl h-[580px] bg-slate-50 dark:bg-neutral-900 rounded-3xl border border-slate-200 dark:border-neutral-800 shadow-2xl overflow-hidden">
        
        {/* Header */}
        <div className="flex items-center justify-between shrink-0 px-6 py-4 bg-white dark:bg-neutral-950 border-b border-slate-100 dark:border-neutral-800/80">
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-50 text-indigo-700 dark:bg-indigo-950/80 dark:text-indigo-300">
              <Sparkles className="h-4 w-4" aria-hidden />
            </span>
            <div>
              <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
                AI Study Flashcards
              </h2>
              <p className="text-xs text-slate-500 dark:text-neutral-400 max-w-[280px] sm:max-w-md truncate">
                Based on notes: <span className="font-medium">{pageTitle || 'Untitled page'}</span>
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close flashcards panel"
            className="flex h-9 w-9 items-center justify-center rounded-xl text-slate-500 transition hover:bg-slate-100 hover:text-slate-800 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100"
          >
            <X className="h-5 w-5" aria-hidden />
          </button>
        </div>

        {/* AI Disclosure Banner */}
        {!loading && !error && totalCards > 0 && (
          <div className="px-6 py-2 bg-indigo-50/50 dark:bg-indigo-950/20 border-b border-indigo-100/50 dark:border-indigo-950/40">
            <AiDisclosureBanner featureKey="rag_notebook" modelLabel="Claude 3.5 Sonnet" />
          </div>
        )}

        {/* Content Body */}
        <div className="flex-1 flex flex-col justify-center px-6 py-6 overflow-y-auto">
          {loading && (
            <div className="flex flex-col items-center justify-center space-y-6 text-center">
              <div className="relative flex items-center justify-center">
                <div className="h-16 w-16 animate-spin rounded-full border-4 border-indigo-200 border-t-indigo-600 dark:border-neutral-700 dark:border-t-indigo-400" />
                <Sparkles className="absolute h-6 w-6 text-indigo-600 dark:text-indigo-400 animate-pulse" />
              </div>
              <div className="space-y-2">
                <p className="text-sm font-semibold text-slate-800 dark:text-neutral-200">
                  Formulating study deck...
                </p>
                <p className="text-xs text-slate-500 dark:text-neutral-400 max-w-xs mx-auto animate-fade-in">
                  Tip: {STUDY_TIPS[tipIndex]}
                </p>
              </div>
            </div>
          )}

          {error && (
            <div className="flex flex-col items-center justify-center space-y-5 text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-rose-100 text-rose-700 dark:bg-rose-950/60 dark:text-rose-300">
                <HelpCircle className="h-6 w-6" aria-hidden />
              </div>
              <div className="space-y-2">
                <p className="text-sm font-semibold text-slate-800 dark:text-neutral-200">
                  Flashcard Generation Failed
                </p>
                <p className="text-xs text-slate-500 dark:text-neutral-400 max-w-md">
                  {error}
                </p>
              </div>
              <button
                type="button"
                onClick={generateFlashcards}
                className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-400"
              >
                <RefreshCw className="h-4 w-4" aria-hidden />
                Try again
              </button>
            </div>
          )}

          {!loading && !error && totalCards === 0 && (
            <div className="flex flex-col items-center justify-center space-y-4 text-center">
              <p className="text-sm text-slate-500 dark:text-neutral-400">
                Write some notes in this notebook first to create flashcards!
              </p>
            </div>
          )}

          {!loading && !error && totalCards > 0 && (
            <>
              {isMastered ? (
                /* Celebration Complete Screen */
                <div className="flex flex-col items-center justify-center space-y-6 text-center animate-fade-in">
                  <div className="relative">
                    <div className="flex h-20 w-20 items-center justify-center rounded-full bg-amber-100 text-amber-600 dark:bg-amber-950/60 dark:text-amber-300 animate-bounce">
                      <Award className="h-10 w-10" aria-hidden />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <h3 className="text-xl font-bold text-slate-900 dark:text-neutral-100">
                      Excellent Job! 🎉
                    </h3>
                    <p className="text-sm text-slate-600 dark:text-neutral-300 max-w-sm">
                      You've mastered all {totalCards} study cards in this notebook page!
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <button
                      type="button"
                      onClick={handleResetStudy}
                      className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700"
                    >
                      <RotateCw className="h-4 w-4" aria-hidden />
                      Study again
                    </button>
                    <button
                      type="button"
                      onClick={generateFlashcards}
                      className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-400"
                    >
                      <RefreshCw className="h-4 w-4" aria-hidden />
                      Regenerate
                    </button>
                  </div>
                </div>
              ) : (
                /* Interactive 3D Flip Card */
                <div className="flex-1 flex flex-col justify-between max-w-md mx-auto w-full">
                  
                  {/* Card Area */}
                  <div 
                    onClick={() => setIsFlipped((prev) => !prev)}
                    className="relative w-full h-[220px] cursor-pointer perspective-1000 group focus:outline-none"
                    tabIndex={0}
                    aria-label={`Flashcard ${currentIndex + 1} of ${totalCards}. Press Space to flip.`}
                  >
                    <div className={`relative w-full h-full duration-500 transform-style-3d ${isFlipped ? 'rotate-y-180' : ''}`}>
                      
                      {/* Front Side */}
                      <div className="absolute inset-0 w-full h-full flex flex-col justify-center items-center p-6 text-center bg-white dark:bg-neutral-950 border-2 border-slate-200 dark:border-neutral-800 rounded-2xl shadow-md backface-hidden group-hover:border-indigo-400 dark:group-hover:border-indigo-500/50 transition-colors overflow-y-auto">
                        <span className="absolute top-3 right-4 text-[10px] uppercase font-bold tracking-wider text-slate-400 dark:text-neutral-500">
                          Front
                        </span>
                        <p className="text-base font-semibold text-slate-800 dark:text-neutral-100 select-text leading-relaxed">
                          {flashcards[currentIndex]?.front}
                        </p>
                        <p className="absolute bottom-3 text-[10px] text-slate-400 dark:text-neutral-500">
                          Click card or press Space to flip
                        </p>
                      </div>

                      {/* Back Side */}
                      <div className="absolute inset-0 w-full h-full flex flex-col justify-center items-center p-6 text-center bg-indigo-50/20 dark:bg-indigo-950/15 border-2 border-indigo-400/70 dark:border-indigo-500/50 rounded-2xl shadow-md backface-hidden rotate-y-180 overflow-y-auto">
                        <span className="absolute top-3 right-4 text-[10px] uppercase font-bold tracking-wider text-indigo-500/80 dark:text-indigo-400/80">
                          Back
                        </span>
                        <p className="text-sm text-slate-800 dark:text-neutral-200 select-text leading-relaxed whitespace-pre-line">
                          {flashcards[currentIndex]?.back}
                        </p>
                        <p className="absolute bottom-3 text-[10px] text-indigo-500/60 dark:text-indigo-400/50">
                          Click card to view front
                        </p>
                      </div>

                    </div>
                  </div>

                  {/* Actions & Navigation Controls */}
                  <div className="mt-6 space-y-4">
                    
                    {/* Mastery Action Button */}
                    <div className="flex justify-center">
                      <button
                        type="button"
                        onClick={handleToggleLearned}
                        className={`inline-flex items-center gap-2 rounded-full px-5 py-1.5 text-xs font-semibold shadow-sm transition ${
                          learned[currentIndex]
                            ? 'bg-emerald-100 text-emerald-800 hover:bg-emerald-200 dark:bg-emerald-950 dark:text-emerald-300 dark:hover:bg-emerald-900/60'
                            : 'bg-slate-200 text-slate-700 hover:bg-slate-300 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700'
                        }`}
                      >
                        <CheckCircle2 className={`h-4 w-4 ${learned[currentIndex] ? 'fill-emerald-500 text-white dark:fill-emerald-400 dark:text-neutral-950' : ''}`} />
                        {learned[currentIndex] ? 'Marked as Learned' : 'Mark as Learned'}
                      </button>
                    </div>

                    {/* Navigation buttons */}
                    <div className="flex items-center justify-between gap-4">
                      <button
                        type="button"
                        onClick={handlePrev}
                        disabled={currentIndex === 0}
                        className="flex h-10 w-10 items-center justify-center rounded-full bg-white border border-slate-200 text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:pointer-events-none disabled:opacity-40 dark:bg-neutral-950 dark:border-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-800"
                        aria-label="Previous card"
                      >
                        <ArrowLeft className="h-5 w-5" aria-hidden />
                      </button>

                      <div className="text-center">
                        <span className="text-xs font-semibold text-slate-500 dark:text-neutral-400">
                          Card {currentIndex + 1} of {totalCards}
                        </span>
                        <span className="hidden sm:inline-block ml-1.5 text-[10px] text-slate-400 dark:text-neutral-500">
                          (Use Left/Right arrows)
                        </span>
                      </div>

                      <button
                        type="button"
                        onClick={handleNext}
                        disabled={currentIndex === totalCards - 1}
                        className="flex h-10 w-10 items-center justify-center rounded-full bg-white border border-slate-200 text-slate-700 shadow-sm transition hover:bg-slate-50 disabled:pointer-events-none disabled:opacity-40 dark:bg-neutral-950 dark:border-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-800"
                        aria-label="Next card"
                      >
                        <ArrowRight className="h-5 w-5" aria-hidden />
                      </button>
                    </div>

                  </div>

                </div>
              )}
            </>
          )}
        </div>

        {/* Footer / Progress Bar */}
        {!loading && !error && totalCards > 0 && (
          <div className="shrink-0 bg-white dark:bg-neutral-950 border-t border-slate-100 dark:border-neutral-800/80 px-6 py-4 flex items-center justify-between gap-4">
            <div className="flex-1">
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs font-semibold text-slate-700 dark:text-neutral-300">
                  Study Deck Mastery Progress
                </span>
                <span className="text-xs font-bold text-indigo-600 dark:text-indigo-400">
                  {percentComplete}% ({learnedCount}/{totalCards})
                </span>
              </div>
              <div className="w-full bg-slate-100 dark:bg-neutral-800 h-2 rounded-full overflow-hidden">
                <div 
                  className="bg-indigo-600 dark:bg-indigo-500 h-full rounded-full motion-safe:transition-[width] motion-safe:duration-300"
                  style={{ width: `${percentComplete}%` }}
                />
              </div>
            </div>

            <button
              type="button"
              onClick={generateFlashcards}
              className="inline-flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm transition hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700 shrink-0"
              title="Regenerate flashcards"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">Regenerate</span>
            </button>
          </div>
        )}

      </div>
    </div>
  )
}
