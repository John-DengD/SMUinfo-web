const assert = require('assert')
const {
  formatDateTime,
  normalizeAnnouncement,
  normalizeAnnouncementList,
  buildAnnouncementPayload
} = require('./announcement-admin')

assert.strictEqual(formatDateTime('2026-06-20T22:53:07'), '2026-06-20 22:53')
assert.strictEqual(formatDateTime(''), '-')

assert.deepStrictEqual(normalizeAnnouncement({
  id: 1,
  title: '公告',
  content: '内容',
  status: 'ACTIVE',
  createdAt: '2026-06-20T22:53:07',
  updatedAt: '2026-06-20T22:54:07'
}), {
  id: 1,
  title: '公告',
  content: '内容',
  status: 'ACTIVE',
  createdAt: '2026-06-20T22:53:07',
  updatedAt: '2026-06-20T22:54:07',
  statusText: '启用中',
  statusClass: 'active',
  actionText: '停用',
  actionStatus: 'INACTIVE',
  actionClass: 'offline',
  createdText: '2026-06-20 22:53',
  updatedText: '2026-06-20 22:54'
})

assert.strictEqual(normalizeAnnouncement({ status: 'INACTIVE' }).actionText, '启用')
assert.strictEqual(normalizeAnnouncement({ status: 'INACTIVE' }).actionClass, 'restore')
assert.strictEqual(normalizeAnnouncementList([{ id: 1 }, { id: 2 }]).length, 2)
assert.deepStrictEqual(buildAnnouncementPayload({
  title: '  校园公告  ',
  content: '  内容  ',
  status: 'OTHER'
}), {
  title: '校园公告',
  content: '内容',
  status: 'INACTIVE'
})

console.log('announcement-admin tests passed')
