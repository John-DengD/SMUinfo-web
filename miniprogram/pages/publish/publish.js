const { request, uploadImage, assetUrl, requireLogin } = require('../../utils/api')

Page({
  data: {
    loading: false,
    categories: [],
    categoryIndex: -1,
    categoryName: '',
    categoryDisplay: '请选择分类',
    conditionOptions: ['全新', '九成新', '八成新', '七成新', '其他'],
    conditionIndex: 1,
    conditionName: '九成新',
    images: [],
    imageUrls: [],
    form: {
      title: '',
      categoryId: '',
      price: '',
      originalPrice: '',
      conditionLevel: '九成新',
      tradeLocation: '',
      description: ''
    }
  },

  onShow() {
    if (!requireLogin('/pages/publish/publish', { replace: true })) return
    if (!this.data.categories.length) this.loadCategories()
  },

  async loadCategories() {
    const categories = await request({ url: '/api/categories', silent: true }).catch(() => [])
    this.setData({ categories })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`form.${field}`]: e.detail.value })
  },

  onCategoryChange(e) {
    const index = Number(e.detail.value)
    const category = this.data.categories[index]
    this.setData({
      categoryIndex: index,
      categoryName: category ? category.name : '',
      categoryDisplay: category ? category.name : '请选择分类',
      'form.categoryId': category ? category.id : ''
    })
  },

  onConditionChange(e) {
    const index = Number(e.detail.value)
    this.setData({
      conditionIndex: index,
      conditionName: this.data.conditionOptions[index],
      'form.conditionLevel': this.data.conditionOptions[index]
    })
  },

  chooseImages() {
    wx.chooseMedia({
      count: 9 - this.data.images.length,
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
    if (!form.title.trim()) return '请输入商品标题'
    if (!form.categoryId) return '请选择分类'
    if (!form.price || Number(form.price) <= 0) return '请输入有效价格'
    return ''
  },

  async submit() {
    if (!requireLogin('/pages/publish/publish', { replace: true })) return
    const form = {
      ...this.data.form,
      title: this.data.form.title.trim(),
      price: Number(this.data.form.price),
      originalPrice: this.data.form.originalPrice ? Number(this.data.form.originalPrice) : null,
      images: this.data.images
    }
    const msg = this.validate(form)
    if (msg) {
      wx.showToast({ title: msg, icon: 'none' })
      return
    }
    this.setData({ loading: true })
    try {
      const product = await request({ url: '/api/products', method: 'POST', data: form, auth: true })
      wx.showToast({ title: '发布成功', icon: 'success' })
      this.resetForm()
      setTimeout(() => {
        wx.navigateTo({ url: `/pages/detail/detail?id=${product.id}` })
      }, 400)
    } finally {
      this.setData({ loading: false })
    }
  },

  resetForm() {
    this.setData({
      categoryIndex: -1,
      categoryName: '',
      categoryDisplay: '请选择分类',
      conditionIndex: 1,
      conditionName: '九成新',
      images: [],
      imageUrls: [],
      form: {
        title: '',
        categoryId: '',
        price: '',
        originalPrice: '',
        conditionLevel: '九成新',
        tradeLocation: '',
        description: ''
      }
    })
  }
})
