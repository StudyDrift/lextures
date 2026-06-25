'use strict';

const { test } = require('node:test');
const assert = require('node:assert/strict');
const { unwrapEnvelope } = require('../lib/api');

test('unwrapEnvelope extracts data payload', () => {
  const out = unwrapEnvelope({
    event_id: '1',
    event_type: 'grade.posted',
    data: { pointsEarned: 10 },
  });
  assert.equal(out.pointsEarned, 10);
  assert.equal(out.event_type, 'grade.posted');
});

test('app exports required triggers and actions', () => {
  const app = require('../index.js');
  assert.ok(app.triggers.new_enrollment);
  assert.ok(app.triggers.grade_posted);
  assert.ok(app.triggers.quiz_completed);
  assert.ok(app.creates.enroll_user);
  assert.ok(app.creates.post_grade);
});
