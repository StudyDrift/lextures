'use strict';

const authHeaders = (bundle) => ({
  Authorization: `Bearer ${bundle.authData.accessToken}`,
  Accept: 'application/json',
  'Content-Type': 'application/json',
});

const unwrapEnvelope = (payload) => {
  if (!payload) return {};
  if (payload.data && typeof payload.data === 'object') {
    return { ...payload.data, event_id: payload.event_id, event_type: payload.event_type };
  }
  return payload;
};

module.exports = { authHeaders, unwrapEnvelope };
