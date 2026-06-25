'use strict';

const { authHeaders } = require('../lib/api');

module.exports = {
  key: 'get_user',
  noun: 'User',
  display: {
    label: 'Get User',
    description: 'Retrieves a user by UUID. Email is included only when your token has the pii:read scope.',
  },
  operation: {
    inputFields: [
      {
        key: 'userId',
        label: 'User ID',
        type: 'string',
        required: true,
      },
    ],
    perform: async (z, bundle) => {
      const response = await z.request({
        url: `${bundle.authData.apiBaseUrl}/api/v1/users/${bundle.inputData.userId}`,
        method: 'GET',
        headers: authHeaders(bundle),
      });
      return response.data;
    },
    sample: {
      id: '00000000-0000-0000-0000-000000000001',
      displayName: 'Alex Example',
    },
  },
};
