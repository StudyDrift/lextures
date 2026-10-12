import assert from 'node:assert/strict'
import { describe, it, afterEach } from 'node:test'
import { boiseDate, filterPublished, isPublishedOn, publishToday } from './publish-date.ts'
import { GUIDES, publishedGuides } from './guides-data.ts'

describe('boiseDate', () => {
  it('uses America/Boise, not UTC', () => {
    // 2026-10-13 05:30 UTC is still 2026-10-12 23:30 in Boise (MDT, UTC-6).
    assert.equal(boiseDate(new Date('2026-10-13T05:30:00Z')), '2026-10-12')
    // 12:00 UTC (6am MDT) is the next Boise day.
    assert.equal(boiseDate(new Date('2026-10-13T12:00:00Z')), '2026-10-13')
    // Winter (MST, UTC-7).
    assert.equal(boiseDate(new Date('2026-12-01T06:59:00Z')), '2026-11-30')
    assert.equal(boiseDate(new Date('2026-12-01T07:00:00Z')), '2026-12-01')
  })
})

describe('isPublishedOn', () => {
  it('publishes on and after the date, not before', () => {
    assert.equal(isPublishedOn('2026-10-13', '2026-10-12'), false)
    assert.equal(isPublishedOn('2026-10-13', '2026-10-13'), true)
    assert.equal(isPublishedOn('2026-10-13', '2026-10-14'), true)
  })
  it('treats a missing date as published and rejects bad dates', () => {
    assert.equal(isPublishedOn(undefined, '2026-10-12'), true)
    assert.throws(() => isPublishedOn('Oct 13', '2026-10-12'))
  })
  it('filters lists', () => {
    const items = [{ id: 'a' }, { id: 'b', publishDate: '2026-10-13' }, { id: 'c', publishDate: '2026-10-15' }]
    assert.deepEqual(filterPublished(items, '2026-10-13').map(i => i.id), ['a', 'b'])
  })
})

describe('publishToday override', () => {
  afterEach(() => { delete process.env.PUBLISH_DATE_OVERRIDE })
  it('honors PUBLISH_DATE_OVERRIDE', () => {
    process.env.PUBLISH_DATE_OVERRIDE = '2026-10-15'
    assert.equal(publishToday(), '2026-10-15')
    process.env.PUBLISH_DATE_OVERRIDE = 'tomorrow'
    assert.throws(() => publishToday())
  })
})

describe('scheduled guides', () => {
  it('releases the homeschool guides on Tuesday and Thursday', () => {
    const slugs = today => publishedGuides(today).map(g => g.slug)
    assert.deepEqual(slugs('2026-10-12'), [])
    assert.deepEqual(slugs('2026-10-13'), ['canvas-lite-for-homeschool'])
    assert.deepEqual(slugs('2026-10-15').sort(), ['canvas-lite-for-homeschool', 'lms-for-homeschool-co-ops'])
  })
  it('every guide has a valid publishDate and 40-60 word FAQ answers', () => {
    for (const guide of GUIDES) {
      assert.match(guide.publishDate, /^\d{4}-\d{2}-\d{2}$/)
      assert.ok(guide.description.length <= 160, guide.slug)
      for (const faq of guide.faqs) {
        const words = faq.answer.split(/\s+/).length
        assert.ok(words >= 25 && words <= 70, `${guide.slug}: ${faq.question} has ${words} words`)
      }
    }
  })
})
