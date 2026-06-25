'use strict';

const makeRestHookTrigger = require('../lib/rest_hook_trigger');

module.exports = makeRestHookTrigger({
  key: 'grade_posted',
  noun: 'Grade',
  label: 'Grade Posted',
  description: 'Triggers when an instructor posts a grade to the gradebook.',
  eventType: 'grade.posted',
  sample: {
    courseId: '00000000-0000-0000-0000-000000000001',
    moduleItemId: '00000000-0000-0000-0000-000000000002',
    studentUserId: '00000000-0000-0000-0000-000000000003',
    pointsEarned: 92.5,
  },
});
