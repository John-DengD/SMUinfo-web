const { request, assetUrl, formatPrice, requireLogin } = require('../../utils/api')

Page({
  data: {
    products: [],
    loading: false,
    emptyVisible: false
  },

  onShow() {
    if (!requireLogin('/pages/favorites/favorites', { replace: true })) return
    this.load()
  },

  async load() {
    this.setData({ loading: true })
    try {
      const data = await request({ url: '/api/favorites', auth: true })
      const products = (data.records || []).map(item => ({
        ...item,
        coverUrl: assetUrl(item.cover),
        priceText: formatPrice(item.price),
        metaText: `${item.conditionLevel || '九成新'} · ${item.tradeLocation || item.sellerCampus || '校园'}`
      }))
      this.setData({
        products,
        emptyVisible: products.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  goDetail(e) {
    wx.navigateTo({ url: `/pages/detail/detail?id=${e.currentTarget.dataset.id}` })
  }
})
