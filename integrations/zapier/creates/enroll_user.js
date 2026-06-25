'use strict';

const { authHeaders } = require('../lib/api');

module.exports = {
  key: 'enroll_user',
  noun: 'Enrollment',
  display: {
    label: 'Enroll User',
    description: 'Enrolls an existing Lextures user in a course by email address.',
  },
  operation: {
    inputFields: [
      {
        key: 'courseId',
        label: 'Course ID',
        type: 'string',
        required: true,
        helpText: 'UUID of the course. Copy from a Get Course step or the Lextures admin UI.',
      },
      {
        key: 'email',
        label: 'User Email',
        type: 'string',
        required: true,
        helpText: 'Email of an existing user in your organization.',
      },
      {
        key: 'courseRole',
        label: 'Course Role',
        type: 'string',
        required: false,
        default: 'student',
        helpText: 'Enrollment role key (defaults to student).',
      },
    ],
    perform: async (z, bundle) => {
      const response = await z.request({
        url: `${bundle.authData.apiBaseUrl}/api/v1/courses/${bundle.inputData.courseId}/enrollments`,
        method: 'POST',
        headers: authHeaders(bundle),
        body: {
          email: bundle.inputData.email,
          courseRole: bundle.inputData.courseRole || 'student',
        },
      });
      return response.data;
    },
    sample: {
      id: '00000000-0000-0000-0000-000000000001',
      userId: '00000000-0000-0000-0000-000000000002',
      role: 'student',
      state: 'active',
    },
  },
};
