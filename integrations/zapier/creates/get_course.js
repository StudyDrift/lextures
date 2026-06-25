'use strict';

const { authHeaders } = require('../lib/api');

module.exports = {
  key: 'get_course',
  noun: 'Course',
  display: {
    label: 'Get Course',
    description: 'Retrieves a course by UUID.',
  },
  operation: {
    inputFields: [
      {
        key: 'courseId',
        label: 'Course ID',
        type: 'string',
        required: true,
      },
    ],
    perform: async (z, bundle) => {
      const response = await z.request({
        url: `${bundle.authData.apiBaseUrl}/api/v1/courses/${bundle.inputData.courseId}`,
        method: 'GET',
        headers: authHeaders(bundle),
      });
      return response.data;
    },
    sample: {
      id: '00000000-0000-0000-0000-000000000001',
      courseCode: 'CS101',
      title: 'Intro to Computing',
      published: true,
    },
  },
};
