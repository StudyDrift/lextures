'use strict';

const makeRestHookTrigger = require('../lib/rest_hook_trigger');

module.exports = makeRestHookTrigger({
  key: 'new_enrollment',
  noun: 'Enrollment',
  label: 'New Enrollment',
  description: 'Triggers when a student is enrolled in a course.',
  eventType: 'enrollment.created',
  sample: {
    enrollmentId: '00000000-0000-0000-0000-000000000001',
    studentUserId: '00000000-0000-0000-0000-000000000002',
    courseId: '00000000-0000-0000-0000-000000000003',
    role: 'student',
  },
});
