function formatDateTime(value) {
  if (!value) return '-'
  const text = String(value)
  return text.replace('T', ' ').slice(0, 16)
}

function normalizeAnnouncement(item) {
  const status = item.status || 'INACTIVE'
  return {
    ...item,
    status,
    statusText: status === 'ACTIVE' ? '启用中' : '已停用',
    statusClass: status === 'ACTIVE' ? 'active' : 'inactive',
    actionText: status === 'ACTIVE' ? '停用' : '启用',
    actionStatus: status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE',
    actionClass: status === 'ACTIVE' ? 'offline' : 'restore',
    createdText: formatDateTime(item.createdAt),
    updatedText: formatDateTime(item.updatedAt)
  }
}

function normalizeAnnouncementList(list) {
  return (list || []).map(normalizeAnnouncement)
}

function buildAnnouncementPayload(form) {
  return {
    title: (form.title || '').trim(),
    content: (form.content || '').trim(),
    status: form.status === 'ACTIVE' ? 'ACTIVE' : 'INACTIVE'
  }
}

module.exports = {
  formatDateTime,
  normalizeAnnouncement,
  normalizeAnnouncementList,
  buildAnnouncementPayload
}
