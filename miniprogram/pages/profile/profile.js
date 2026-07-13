const { request, getUser, requireLogin } = require('../../utils/api')

const namePattern = /^[\u4e00-\u9fa5A-Za-z·.\- ]+$/

Page({
  data: {
    loading: false,
    form: {
      name: '',
      studentNo: '',
      phone: '',
      college: '',
      campus: ''
    }
  },

  onShow() {
    if (!requireLogin('/pages/profile/profile', { replace: true })) return
    const user = getUser()
    if (user) this.fill(user)
    this.loadMe()
  },

  async loadMe() {
    const user = await request({ url: '/api/users/me', auth: true, silent: true }).catch(() => null)
    if (user) {
      wx.setStorageSync('user', user)
      this.fill(user)
    }
  },

  fill(user) {
    this.setData({
      form: {
        name: user.name || '',
        studentNo: user.studentNo || '',
        phone: user.phone || '',
        college: user.college || '',
        campus: user.campus || ''
      }
    })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`form.${field}`]: e.detail.value })
  },

  validate(form) {
    const name = form.name.trim().replace(/\s+/g, ' ')
    if (!name) return '请输入姓名'
    if (name.length < 2 || name.length > 20) return '姓名长度需为 2-20 个字符'
    if (!namePattern.test(name)) return '姓名不能包含数字或特殊字符'
    return ''
  },

  async save() {
    const form = {
      ...this.data.form,
      name: this.data.form.name.trim().replace(/\s+/g, ' ')
    }
    const msg = this.validate(form)
    if (msg) {
      wx.showToast({ title: msg, icon: 'none' })
      return
    }
    this.setData({ loading: true })
    try {
      const user = await request({
        url: '/api/users/me',
        method: 'PUT',
        auth: true,
        data: {
          name: form.name,
          phone: form.phone,
          college: form.college,
          campus: form.campus
        }
      })
      wx.setStorageSync('user', user)
      wx.showToast({ title: '已保存', icon: 'success' })
      setTimeout(() => wx.navigateBack(), 350)
    } finally {
      this.setData({ loading: false })
    }
  }
})
