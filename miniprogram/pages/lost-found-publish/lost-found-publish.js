const { request, uploadImage, assetUrl, requireLogin } = require('../../utils/api')

Page({
  data: {
    loading: false,
    typeOptions: ['我丢了', '我捡到'],
    typeValues: ['LOST', 'FOUND'],
    typeIndex: 0,
    typeDisplay: '我丢了',
    images: [],
    imageUrls: [],
    form: {
      type: 'LOST',
      title: '',
      location: '',
      contact: '',
      description: ''
    }
  },

  onShow() {
    requireLogin('/pages/lost-found-publish/lost-found-publish', { replace: true })
  },

  onTypeChange(e) {
    const index = Number(e.detail.value)
    this.setData({
      typeIndex: index,
      typeDisplay: this.data.typeOptions[index],
      'form.type': this.data.typeValues[index]
    })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`form.${field}`]: e.detail.value })
  },

  chooseImages() {
    wx.chooseMedia({
      count: 6 - this.data.images.length,
      mediaType: ['image'],
      sourceType: ['album', 'camera'],
      success: async (res) => {
        wx.showLoading({ title: '上传中' })
        try {
          const images = this.data.images.slice()
          for (const file of res.tempFiles) {
            const data = await uploadImage(file.tempFilePath)
            images.push(data.url)
          }
          this.setData({
            images,
            imageUrls: images.map(assetUrl)
          })
        } finally {
          wx.hideLoading()
        }
      }
    })
  },

  removeImage(e) {
    const index = e.currentTarget.dataset.index
    const images = this.data.images.slice()
    images.splice(index, 1)
    this.setData({
      images,
      imageUrls: images.map(assetUrl)
    })
  },

  validate(form) {
    if (!form.title.trim()) return '请输入标题'
    if (!form.description.trim()) return '请输入详细描述'
    return ''
  },

  async submit() {
    if (!requireLogin('/pages/lost-found-publish/lost-found-publish', { replace: true })) return
    const form = {
      ...this.data.form,
      title: this.data.form.title.trim(),
      description: this.data.form.description.trim(),
      location: this.data.form.location.trim(),
      contact: this.data.form.contact.trim(),
      images: this.data.images
    }
    const message = this.validate(form)
    if (message) {
      wx.showToast({ title: message, icon: 'none' })
      return
    }
    this.setData({ loading: true })
    try {
      const item = await request({ url: '/api/lost-found', method: 'POST', auth: true, data: form })
      wx.showToast({ title: '发布成功', icon: 'success' })
      setTimeout(() => {
        wx.redirectTo({ url: `/pages/lost-found-detail/lost-found-detail?id=${item.id}` })
      }, 450)
    } finally {
      this.setData({ loading: false })
    }
  }
})
