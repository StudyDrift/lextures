'use strict';

const { authHeaders, unwrapEnvelope } = require('../lib/api');

const makeRestHookTrigger = ({ key, noun, label, description, eventType, sample }) => ({
  key,
  noun,
  display: { label, description },
  operation: {
    type: 'hook',
    perform: async (z, bundle) => [unwrapEnvelope(bundle.cleanedRequest)],
    performSubscribe: async (z, bundle) => {
      const response = await z.request({
        url: `${bundle.authData.apiBaseUrl}/api/v1/webhooks`,
        method: 'POST',
        headers: authHeaders(bundle),
        body: {
          label: `Zapier: ${label}`,
          endpointUrl: bundle.targetUrl,
          eventTypes: [eventType],
          settings: { source: 'zapier' },
        },
      });
      return response.data.subscription;
    },
    performUnsubscribe: async (z, bundle) => {
      await z.request({
        url: `${bundle.authData.apiBaseUrl}/api/v1/webhooks/${bundle.subscribeData.id}`,
        method: 'DELETE',
        headers: authHeaders(bundle),
      });
      return {};
    },
    performList: async () => [],
    sample: { ...sample, event_type: eventType },
  },
});

module.exports = makeRestHookTrigger;
