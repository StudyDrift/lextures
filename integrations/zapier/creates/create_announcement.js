'use strict';

const { authHeaders } = require('../lib/api');

module.exports = {
  key: 'create_announcement',
  noun: 'Announcement',
  display: {
    label: 'Create Announcement',
    description: 'Posts an announcement to a course announcements channel.',
  },
  operation: {
    inputFields: [
      {
        key: 'courseId',
        label: 'Course ID',
        type: 'string',
        required: true,
      },
      {
        key: 'title',
        label: 'Title',
        type: 'string',
        required: true,
      },
      {
        key: 'body',
        label: 'Body',
        type: 'text',
        required: true,
      },
    ],
    perform: async (z, bundle) => {
      const response = await z.request({
        url: `${bundle.authData.apiBaseUrl}/api/v1/courses/${bundle.inputData.courseId}/announcements`,
        method: 'POST',
        headers: authHeaders(bundle),
        body: {
          title: bundle.inputData.title,
          body: bundle.inputData.body,
        },
      });
      return response.data;
    },
    sample: { id: '00000000-0000-0000-0000-000000000001' },
  },
};
