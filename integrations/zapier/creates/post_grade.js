'use strict';

const { authHeaders } = require('../lib/api');

module.exports = {
  key: 'post_grade',
  noun: 'Grade',
  display: {
    label: 'Post Grade',
    description: 'Writes a gradebook cell and optionally posts it to the learner.',
  },
  operation: {
    inputFields: [
      { key: 'courseId', label: 'Course ID', type: 'string', required: true },
      { key: 'studentUserId', label: 'Student User ID', type: 'string', required: true },
      { key: 'moduleItemId', label: 'Assignment/Module Item ID', type: 'string', required: true },
      { key: 'pointsEarned', label: 'Points Earned', type: 'number', required: true },
      {
        key: 'post',
        label: 'Post to Student',
        type: 'boolean',
        required: false,
        default: 'true',
        helpText: 'When true, releases the grade to the learner immediately.',
      },
    ],
    perform: async (z, bundle) => {
      const response = await z.request({
        url: `${bundle.authData.apiBaseUrl}/api/v1/grades`,
        method: 'POST',
        headers: authHeaders(bundle),
        body: {
          courseId: bundle.inputData.courseId,
          studentUserId: bundle.inputData.studentUserId,
          moduleItemId: bundle.inputData.moduleItemId,
          pointsEarned: Number(bundle.inputData.pointsEarned),
          post: bundle.inputData.post !== false && bundle.inputData.post !== 'false',
        },
      });
      return response.data;
    },
    sample: {
      courseId: '00000000-0000-0000-0000-000000000001',
      studentUserId: '00000000-0000-0000-0000-000000000002',
      moduleItemId: '00000000-0000-0000-0000-000000000003',
      pointsEarned: '92',
    },
  },
};
