const { request, setSession } = require('../../utils/api')

Page({
  data: {
    redirect: '',
    loading: false,
    form: {
      studentNo: '',
      password: ''
    }
  },

  onLoad(options) {
    this.setData({ redirect: options.redirect || '' })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`form.${field}`]: e.detail.value })
  },

  async submit() {
    const form = {
      studentNo: this.data.form.studentNo.trim(),
      password: this.data.form.password
    }
    if (!form.studentNo || !form.password) {
      wx.showToast({ title: '请填写学号和密码', icon: 'none' })
      return
    }
    this.setData({ loading: true })
    try {
      const data = await request({ url: '/api/auth/login', method: 'POST', data: form })
      setSession(data.token, data.user)
      wx.showToast({ title: '登录成功', icon: 'success' })
      setTimeout(() => {
        this.goAfterLogin()
      }, 350)
    } finally {
      this.setData({ loading: false })
    }
  },

  goRegister() {
    const redirect = this.data.redirect ? `?redirect=${encodeURIComponent(this.data.redirect)}` : ''
    wx.navigateTo({ url: `/pages/register/register${redirect}` })
  },

  goAfterLogin() {
    const target = this.data.redirect ? decodeURIComponent(this.data.redirect) : '/pages/me/me'
    const rootPages = ['/pages/home/home', '/pages/me/me']
    if (rootPages.includes(target)) {
      wx.reLaunch({ url: target })
      return
    }
    wx.redirectTo({ url: target })
  }
})
