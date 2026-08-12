import assert from 'node:assert/strict'
import test from 'node:test'
import { checkFamily, checkUtilityPage, similarity } from './utility-check.mjs'
const prose = Array.from({ length: 160 }, (_, i) => `specific${i}`).join(' ')
const valid = { family: 'glossary', path: '/glossary/example', prose, actions: ['copy'], sources: ['primary'], inboundLinks: 3, outboundLinks: ['/a', '/b', '/c'], reviewRequired: true, reviewedBy: 'reviewer' }
test('passes all four tests', () => assert.equal(checkUtilityPage(valid, [valid]).indexed, true))
test('missing review or links forces noindex', () => assert.equal(checkUtilityPage({ ...valid, reviewedBy: '', inboundLinks: 2 }, [valid]).robots, 'noindex,follow'))
test('detects duplicate prose', () => assert.equal(similarity(prose, prose), 1))
test('requires eighty percent family floor', () => assert.equal(checkFamily([valid, { ...valid, path: '/bad', sources: [] }]).launchable, false))
