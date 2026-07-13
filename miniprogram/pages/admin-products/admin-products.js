const { request, assetUrl, formatPrice, requireLogin, getUser } = require('../../utils/api')

Page({
  data: {
    keyword: '',
    products: [],
    page: 1,
    size: 20,
    total: 0,
    loading: false,
    emptyVisible: false,
    operatingId: ''
  },

  onShow() {
    if (!requireLogin('/pages/admin-products/admin-products', { replace: true })) return
    const user = getUser()
    if (!user || user.role !== 'ADMIN') {
      wx.showToast({ title: '仅管理员可访问', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 400)
      return
    }
    this.load(true)
  },

  onInput(e) {
    this.setData({ keyword: e.detail.value })
  },

  onSearch() {
    this.load(true)
  },

  async load(reset) {
    if (reset) this.setData({ page: 1 })
    this.setData({ loading: true })
    try {
      const params = {
        page: this.data.page,
        size: this.data.size
      }
      const keyword = this.data.keyword.trim()
      if (keyword) params.keyword = keyword
      const data = await request({ url: '/api/admin/products', auth: true, data: params })
      const records = (data.records || []).map(item => ({
        ...item,
        coverUrl: assetUrl(item.cover),
        priceText: formatPrice(item.price),
        statusText: this.statusText(item.status),
        actionText: item.status === 'OFFLINE' ? '恢复' : '下架',
        actionStatus: item.status === 'OFFLINE' ? 'ON_SALE' : 'OFFLINE',
        actionClass: item.status === 'OFFLINE' ? 'restore' : 'offline',
        metaText: `${this.statusText(item.status)} · ${item.sellerName || '同学'} · ${item.categoryName || '未分类'}`
      }))
      const products = reset ? records : this.data.products.concat(records)
      this.setData({
        products,
        total: data.total || 0,
        page: this.data.page + 1,
        emptyVisible: products.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  onReachBottom() {
    if (!this.data.loading && this.data.products.length < this.data.total) this.load(false)
  },

  statusText(status) {
    const map = { ON_SALE: '在售', RESERVED: '已预约', SOLD: '已售出', OFFLINE: '已下架' }
    return map[status] || '未知'
  },

  goDetail(e) {
    wx.navigateTo({ url: `/pages/detail/detail?id=${e.currentTarget.dataset.id}` })
  },

  changeStatus(e) {
    const { id, status } = e.currentTarget.dataset
    wx.showModal({
      title: '商品状态',
      content: status === 'OFFLINE' ? '确认下架这个商品？' : '确认恢复为在售？',
      success: async (res) => {
        if (!res.confirm) return
        this.setData({ operatingId: id })
        try {
          await request({
            url: `/api/admin/products/${id}/status`,
            method: 'PUT',
            auth: true,
            data: { status }
          })
          wx.showToast({ title: '已更新', icon: 'success' })
          this.load(true)
        } finally {
          this.setData({ operatingId: '' })
        }
      }
    })
  }
})
