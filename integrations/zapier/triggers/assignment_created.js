'use strict';

const makeRestHookTrigger = require('../lib/rest_hook_trigger');

module.exports = makeRestHookTrigger({
  key: 'assignment_created',
  noun: 'Assignment',
  label: 'Assignment Created',
  description: 'Triggers when a new assignment is published in a course.',
  eventType: 'assignment.created',
  sample: {
    courseId: '00000000-0000-0000-0000-000000000001',
    structureItemId: '00000000-0000-0000-0000-000000000002',
    title: 'Week 3 Essay',
  },
});
