'use strict';

const testAuth = async (z, bundle) => {
  const response = await z.request({
    url: `${bundle.authData.apiBaseUrl}/api/v1/me`,
    method: 'GET',
  });
  return response.data;
};

module.exports = {
  type: 'custom',
  test: testAuth,
  connectionLabel: '{{displayName}} ({{email}})',
  fields: [
    {
      key: 'apiBaseUrl',
      label: 'Lextures API Base URL',
      type: 'string',
      required: true,
      helpText: 'Your Lextures instance URL, e.g. https://app.lextures.com or http://localhost:8080 for local testing.',
      default: 'https://app.lextures.com',
    },
    {
      key: 'accessToken',
      label: 'Personal Access Token',
      type: 'password',
      required: true,
      helpText: 'Create a token in Lextures under Settings → Integrations → API Keys. Grant scopes needed for your Zaps.',
    },
  ],
};
