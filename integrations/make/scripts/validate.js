#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');

const root = path.join(__dirname, '..');
const jsonFiles = [
  path.join(root, 'app', 'base.json'),
  ...fs.readdirSync(path.join(root, 'modules')).map((f) => path.join(root, 'modules', f)),
];

let failed = false;
for (const file of jsonFiles) {
  try {
    const raw = fs.readFileSync(file, 'utf8');
    JSON.parse(raw);
    console.log(`OK ${path.relative(root, file)}`);
  } catch (err) {
    failed = true;
    console.error(`FAIL ${path.relative(root, file)}: ${err.message}`);
  }
}

if (failed) process.exit(1);
