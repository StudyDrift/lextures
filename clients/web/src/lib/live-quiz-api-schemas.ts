import { z } from 'zod'

export const quizKitSchema = z.object({
  id: z.string(),
  courseId: z.string(),
  title: z.string(),
  description: z.string(),
  slug: z.string(),
  coverImageRef: z.string().nullable().optional(),
  status: z.enum(['draft', 'ready', 'archived']),
  visibility: z.enum(['private', 'course', 'org', 'public']),
  tags: z.array(z.string()),
  questionCount: z.number(),
  archived: z.boolean(),
  createdBy: z.string().nullable().optional(),
  createdAt: z.string(),
  updatedAt: z.string(),
})

export const listKitsResponseSchema = z.object({
  kits: z.array(quizKitSchema),
  total: z.number(),
  page: z.number(),
  pageSize: z.number(),
  totalPages: z.number(),
})

export const liveQuizQuestionTypeSchema = z.enum([
  'mc_single',
  'mc_multiple',
  'true_false',
  'type_answer',
  'numeric',
  'poll',
  'ordering',
  'word_cloud',
])

export const liveQuizPointsStyleSchema = z.enum(['standard', 'double', 'no_points'])

export const liveQuizOptionSchema = z.object({
  id: z.string(),
  text: z.string(),
  mediaRef: z.string().nullable().optional(),
  mediaAlt: z.string().nullable().optional(),
  isCorrect: z.boolean(),
})

export const liveQuizQuestionSchema = z.object({
  id: z.string(),
  kitId: z.string(),
  position: z.number(),
  questionType: liveQuizQuestionTypeSchema,
  prompt: z.string(),
  promptMediaRef: z.string().nullable().optional(),
  promptMediaAlt: z.string().nullable().optional(),
  options: z.array(liveQuizOptionSchema),
  correctAnswer: z.unknown().nullable().optional(),
  timeLimitSeconds: z.number(),
  pointsStyle: liveQuizPointsStyleSchema,
  answerShuffle: z.boolean(),
  explanation: z.string().nullable().optional(),
  sourceQuestionId: z.string().nullable().optional(),
  version: z.number(),
  createdAt: z.string(),
  updatedAt: z.string(),
})

export const listQuestionsResponseSchema = z.object({
  questions: z.array(liveQuizQuestionSchema),
})

export const validateKitResponseSchema = z.object({
  isReady: z.boolean(),
  issues: z.array(
    z.object({
      questionId: z.string(),
      code: z.string(),
      message: z.string(),
    }),
  ),
})

export const bankCandidateSchema = z.object({
  id: z.string(),
  questionType: z.string(),
  stem: z.string(),
  status: z.string(),
})
