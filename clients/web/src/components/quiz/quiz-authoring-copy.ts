export type QuizAuthoringCopy = {
  family: boolean
  buildWithAi: string
  buildWithAiDetail: string | null
  buildWithAiDescription: string
  buildWithAiPlaceholder: string
  buildWithAiSubmitHint: string
  suggestQuestions: string
  suggestQuestionsDetail: string | null
  suggestQuestionsBody: string
  suggestQuestionsPromptLabel: string
  suggestQuestionsPlaceholder: string
  suggestQuestionsAction: string
  suggestQuestionsBusy: string
  suggestQuestionsAdvanced: string | null
  lockdownSummary: string
  lockdownSummaryDetail: string | null
  lockdownSettingsLabel: string
  lockdownSettingsHint: string
  lockdownAdaptiveHint: string
  lockdownOffNote: (modeLabel: string) => string
  focusLossSummary: string
  focusLossLabel: string
  focusLossHint: string
  lockdownOptionStandard: string
  lockdownOptionOneAtATime: string
  lockdownOptionKiosk: string
  advancedLockdown: string | null
}

const campus: QuizAuthoringCopy = {
  family: false,
  buildWithAi: 'Build with AI',
  buildWithAiDetail: null,
  buildWithAiDescription:
    'Describe what this quiz intro should cover. The draft replaces the current editor content; nothing is saved until you click Save.',
  buildWithAiPlaceholder:
    'e.g. Intro for a midterm on cell division: overview, time limit note, and what students should review…',
  buildWithAiSubmitHint: '⌘/Ctrl + Enter to generate',
  suggestQuestions: 'Generate questions',
  suggestQuestionsDetail: null,
  suggestQuestionsBody:
    'Describe the topic or learning goals. The model will create the requested number of questions using the quiz question types (multiple choice, true/false, fill-in-the-blank, short answer, and essay). Type @ to tag a content page or assignment — it appears as a highlighted @name; the item’s body is pulled in when you click Generate.',
  suggestQuestionsPromptLabel: 'Prompt',
  suggestQuestionsPlaceholder:
    'e.g. Five questions on cell division… Type @ to tag a content page or assignment (content is added when you generate).',
  suggestQuestionsAction: 'Generate',
  suggestQuestionsBusy: 'Generating…',
  suggestQuestionsAdvanced: null,
  lockdownSummary: 'Course lockdown feature',
  lockdownSummaryDetail: null,
  lockdownSettingsLabel: 'Lockdown delivery',
  lockdownSettingsHint: 'Kiosk mode requests full-screen, logs tab or window changes, and disables hints.',
  lockdownAdaptiveHint:
    'Adaptive quizzes cannot use server-enforced lockdown. Turn off adaptive generation to enable these modes.',
  lockdownOffNote: (modeLabel) =>
    `This quiz is set to ${modeLabel.toLowerCase()}, but the course lockdown feature is off, so learners currently get standard delivery.`,
  focusLossSummary: 'Focus-loss threshold',
  focusLossLabel: 'Focus-loss flag threshold',
  focusLossHint:
    'Leave empty so attempts are never auto-flagged. When set, exceeding this many logged events marks the attempt for review on submit.',
  lockdownOptionStandard: 'Standard',
  lockdownOptionOneAtATime: 'One question at a time (server enforced)',
  lockdownOptionKiosk: 'Kiosk (fullscreen + focus logging)',
  advancedLockdown: null,
}

const family: QuizAuthoringCopy = {
  family: true,
  buildWithAi: 'Help write the intro',
  buildWithAiDetail: 'Also called Build with AI',
  buildWithAiDescription:
    'Describe what learners should read before they start. The draft replaces the introduction. Nothing is saved until you click Save.',
  buildWithAiPlaceholder:
    'e.g. A short note before a quiz on cell division: how long it takes and what to review…',
  buildWithAiSubmitHint: '⌘/Ctrl + Enter to draft',
  suggestQuestions: 'Suggest questions',
  suggestQuestionsDetail: 'Also called Generate questions',
  suggestQuestionsBody:
    'Describe the topic or what you want learners to practice. Lextures will suggest that many questions (multiple choice, true/false, fill-in-the-blank, short answer, and essay). Type @ to include a page or assignment from this course.',
  suggestQuestionsPromptLabel: 'What should the questions cover?',
  suggestQuestionsPlaceholder: 'e.g. Five questions on cell division… Type @ to include a page or assignment.',
  suggestQuestionsAction: 'Suggest',
  suggestQuestionsBusy: 'Suggesting…',
  suggestQuestionsAdvanced: 'Advanced name: Generate questions.',
  lockdownSummary: 'Keep the quiz locked while taking',
  lockdownSummaryDetail: 'Also called Course lockdown feature',
  lockdownSettingsLabel: 'Keep the quiz locked while taking',
  lockdownSettingsHint: 'Full screen asks the learner to stay on the quiz and notes when they switch away.',
  lockdownAdaptiveHint: 'Quizzes that change with each answer cannot use this lock. Turn that off to use it.',
  lockdownOffNote: (modeLabel) =>
    `This quiz is set to ${modeLabel}, but keeping the quiz locked while taking is turned off, so learners get the usual experience.`,
  focusLossSummary: 'Flag after leaving the quiz',
  focusLossLabel: 'Flag after leaving the quiz',
  focusLossHint:
    'Leave empty to never flag automatically. When set, leaving the quiz this many times marks the attempt for you to review.',
  lockdownOptionStandard: 'Usual',
  lockdownOptionOneAtATime: 'One question at a time',
  lockdownOptionKiosk: 'Full screen',
  advancedLockdown:
    'Advanced names: Course lockdown feature. Delivery modes are Standard, One at a time, and Kiosk.',
}

export function quizAuthoringCopy(familyAudience: boolean): QuizAuthoringCopy {
  return familyAudience ? family : campus
}
