const { request, assetUrl, formatPrice, requireLogin, getUser } = require('../../utils/api')

Page({
  data: {
    products: [],
    loading: false,
    emptyVisible: false,
    operatingId: ''
  },

  onShow() {
    if (!requireLogin('/pages/my-products/my-products', { replace: true })) return
    this.load()
  },

  async load() {
    const user = getUser()
    if (!user) return
    this.setData({ loading: true })
    try {
      const data = await request({
        url: '/api/products',
        data: {
          sellerId: user.id,
          size: 50
        },
        auth: true
      })
      const products = (data.records || []).map(item => ({
        ...item,
        coverUrl: assetUrl(item.cover),
        priceText: formatPrice(item.price),
        statusText: this.statusText(item.status),
        actionText: item.status === 'OFFLINE' ? '恢复在售' : '下架',
        actionStatus: item.status === 'OFFLINE' ? 'ON_SALE' : 'OFFLINE',
        actionClass: item.status === 'OFFLINE' ? 'restore' : 'offline',
        metaText: `${this.statusText(item.status)} · ${item.tradeLocation || item.sellerCampus || '校园'}`
      }))
      this.setData({
        products,
        emptyVisible: products.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
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
    const text = status === 'OFFLINE' ? '确认下架这个商品？' : '确认恢复为在售？'
    wx.showModal({
      title: '商品状态',
      content: text,
      success: async (res) => {
        if (!res.confirm) return
        this.setData({ operatingId: id })
        try {
          if (status === 'OFFLINE') {
            await request({ url: `/api/products/${id}`, method: 'DELETE', auth: true })
          } else {
            await request({ url: `/api/products/${id}`, method: 'PUT', auth: true, data: { status } })
          }
          wx.showToast({ title: '已更新', icon: 'success' })
          this.load()
        } finally {
          this.setData({ operatingId: '' })
        }
      }
    })
  }
})
