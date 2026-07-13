const { request } = require('../../utils/api')

const fakeStudentNos = ['000000000000', '111111111111', '123456789012']
const namePattern = /^[\u4e00-\u9fa5A-Za-z·.\- ]+$/
const studentNoPattern = /^\d{12}$/

Page({
  data: {
    loading: false,
    redirect: '',
    form: {
      name: '',
      studentNo: '',
      password: '',
      college: '',
      campus: ''
    }
  },

  onLoad(options) {
    this.setData({ redirect: options.redirect || '' })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`form.${field}`]: e.detail.value })
  },

  validate(form) {
    if (!form.name) return '请输入姓名'
    if (form.name.length < 2 || form.name.length > 20) return '姓名长度需为 2-20 个字符'
    if (!namePattern.test(form.name)) return '姓名不能包含数字或特殊字符'
    if (!studentNoPattern.test(form.studentNo)) return '学号必须是 12 位纯数字'
    if (fakeStudentNos.includes(form.studentNo)) return '请填写真实学号'
    if (!form.password || form.password.length < 6) return '密码至少 6 位'
    return ''
  },

  async submit() {
    const form = {
      ...this.data.form,
      name: this.data.form.name.trim().replace(/\s+/g, ' '),
      studentNo: this.data.form.studentNo.trim()
    }
    const msg = this.validate(form)
    if (msg) {
      wx.showToast({ title: msg, icon: 'none' })
      return
    }
    this.setData({ loading: true })
    try {
      await request({ url: '/api/auth/register', method: 'POST', data: form })
      wx.showToast({ title: '注册成功', icon: 'success' })
      setTimeout(() => {
        const redirect = this.data.redirect ? `?redirect=${encodeURIComponent(this.data.redirect)}` : ''
        wx.redirectTo({ url: `/pages/login/login${redirect}` })
      }, 450)
    } finally {
      this.setData({ loading: false })
    }
  }
})
