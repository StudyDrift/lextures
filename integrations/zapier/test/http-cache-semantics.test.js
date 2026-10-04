'use strict';

const { test } = require('node:test');
const assert = require('node:assert/strict');
const CachePolicy = require('http-cache-semantics');

test('shared max-stale does not revive a Set-Cookie entry', () => {
  const now = Date.now();
  const policy = new CachePolicy(
    { headers: {} },
    {
      status: 200,
      headers: {
        'cache-control': 'max-age=1',
        'set-cookie': 'session=secret',
      },
    },
    { shared: true, now: now - 10_000 },
  );
  const result = policy.evaluateRequest({
    headers: { 'cache-control': 'max-stale=99999' },
  });
  assert.equal(result.response, undefined);
});

test('max-stale can still serve an ordinary public stale response', () => {
  const now = Date.now();
  const policy = new CachePolicy(
    { headers: {} },
    {
      status: 200,
      headers: { 'cache-control': 'public, max-age=1' },
    },
    { shared: true, now: now - 10_000 },
  );
  const result = policy.evaluateRequest({
    headers: { 'cache-control': 'max-stale=99999' },
  });
  assert.ok(result.response);
});
