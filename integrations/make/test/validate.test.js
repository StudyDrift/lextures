'use strict';

const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

test('make modules parse as JSON', () => {
  const modulesDir = path.join(__dirname, '..', 'modules');
  for (const file of fs.readdirSync(modulesDir)) {
    JSON.parse(fs.readFileSync(path.join(modulesDir, file), 'utf8'));
  }
  assert.ok(true);
});
