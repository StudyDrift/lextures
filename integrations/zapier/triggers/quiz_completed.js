'use strict';

const makeRestHookTrigger = require('../lib/rest_hook_trigger');

module.exports = makeRestHookTrigger({
  key: 'quiz_completed',
  noun: 'Quiz',
  label: 'Quiz Completed',
  description: 'Triggers when a learner submits a quiz attempt.',
  eventType: 'quiz.completed',
  sample: {
    courseId: '00000000-0000-0000-0000-000000000001',
    moduleItemId: '00000000-0000-0000-0000-000000000002',
    attemptId: '00000000-0000-0000-0000-000000000003',
    studentUserId: '00000000-0000-0000-0000-000000000004',
    pointsEarned: 8,
    scorePercent: 80,
  },
});
