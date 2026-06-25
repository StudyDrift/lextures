'use strict';

const makeRestHookTrigger = require('../lib/rest_hook_trigger');

module.exports = makeRestHookTrigger({
  key: 'assignment_submitted',
  noun: 'Submission',
  label: 'Assignment Submitted',
  description: 'Triggers when a learner submits an assignment.',
  eventType: 'assignment.submitted',
  sample: {
    courseId: '00000000-0000-0000-0000-000000000001',
    moduleItemId: '00000000-0000-0000-0000-000000000002',
    submissionId: '00000000-0000-0000-0000-000000000003',
    submittedBy: '00000000-0000-0000-0000-000000000004',
  },
});
