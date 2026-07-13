const { request, requireLogin, getUser } = require('../../utils/api')
const {
  normalizeAnnouncementList,
  buildAnnouncementPayload
} = require('../../utils/announcement-admin')

Page({
  data: {
    announcements: [],
    loading: false,
    saving: false,
    operatingId: '',
    emptyVisible: false,
    formVisible: false,
    formMode: 'create',
    formTitle: '发布公告',
    form: {
      id: '',
      title: '',
      content: '',
      status: 'ACTIVE'
    },
    activeChecked: true
  },

  onShow() {
    if (!requireLogin('/pages/admin-announcements/admin-announcements', { replace: true })) return
    const user = getUser()
    if (!user || user.role !== 'ADMIN') {
      wx.showToast({ title: '仅管理员可访问', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 400)
      return
    }
    this.load()
  },

  async load() {
    this.setData({ loading: true })
    try {
      const data = await request({ url: '/api/admin/announcements', auth: true })
      const announcements = normalizeAnnouncementList(data || [])
      this.setData({
        announcements,
        emptyVisible: announcements.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  openCreate() {
    this.setData({
      formVisible: true,
      formMode: 'create',
      formTitle: '发布公告',
      form: {
        id: '',
        title: '',
        content: '',
        status: 'ACTIVE'
      },
      activeChecked: true
    })
  },

  openEdit(e) {
    const id = Number(e.currentTarget.dataset.id)
    const item = this.data.announcements.find(row => row.id === id)
    if (!item) return
    this.setData({
      formVisible: true,
      formMode: 'edit',
      formTitle: '编辑公告',
      form: {
        id: item.id,
        title: item.title || '',
        content: item.content || '',
        status: item.status || 'INACTIVE'
      },
      activeChecked: item.status === 'ACTIVE'
    })
  },

  closeForm() {
    if (this.data.saving) return
    this.setData({ formVisible: false })
  },

  noop() {},

  onTitleInput(e) {
    this.setData({ 'form.title': e.detail.value })
  },

  onContentInput(e) {
    this.setData({ 'form.content': e.detail.value })
  },

  onStatusChange(e) {
    const activeChecked = !!e.detail.value
    this.setData({
      activeChecked,
      'form.status': activeChecked ? 'ACTIVE' : 'INACTIVE'
    })
  },

  async submit() {
    const payload = buildAnnouncementPayload(this.data.form)
    if (!payload.title) {
      wx.showToast({ title: '请输入标题', icon: 'none' })
      return
    }
    if (!payload.content) {
      wx.showToast({ title: '请输入内容', icon: 'none' })
      return
    }

    this.setData({ saving: true })
    try {
      if (this.data.formMode === 'edit') {
        await request({
          url: `/api/admin/announcements/${this.data.form.id}`,
          method: 'PUT',
          auth: true,
          data: payload
        })
      } else {
        await request({
          url: '/api/admin/announcements',
          method: 'POST',
          auth: true,
          data: payload
        })
      }
      wx.showToast({ title: '已保存', icon: 'success' })
      this.setData({ formVisible: false })
      this.load()
    } finally {
      this.setData({ saving: false })
    }
  },

  changeStatus(e) {
    const { id, status, title } = e.currentTarget.dataset
    const item = this.data.announcements.find(row => String(row.id) === String(id))
    if (!item) return
    wx.showModal({
      title: '公告状态',
      content: `确认${status === 'ACTIVE' ? '启用' : '停用'}「${title || '这条公告'}」？`,
      success: async (res) => {
        if (!res.confirm) return
        this.setData({ operatingId: id })
        try {
          await request({
            url: `/api/admin/announcements/${id}`,
            method: 'PUT',
            auth: true,
            data: {
              title: item.title,
              content: item.content,
              status
            }
          })
          wx.showToast({ title: '已更新', icon: 'success' })
          this.load()
        } finally {
          this.setData({ operatingId: '' })
        }
      }
    })
  },

  remove(e) {
    const { id, title } = e.currentTarget.dataset
    wx.showModal({
      title: '删除公告',
      content: `确认删除「${title || '这条公告'}」？`,
      confirmColor: '#d21f3c',
      success: async (res) => {
        if (!res.confirm) return
        this.setData({ operatingId: id })
        try {
          await request({
            url: `/api/admin/announcements/${id}`,
            method: 'DELETE',
            auth: true
          })
          wx.showToast({ title: '已删除', icon: 'success' })
          this.load()
        } finally {
          this.setData({ operatingId: '' })
        }
      }
    })
  }
})
