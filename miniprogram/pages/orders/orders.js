const { request, assetUrl, formatPrice, requireLogin, getUser } = require('../../utils/api')

Page({
  data: {
    role: 'buyer',
    buyerTabClass: 'active',
    sellerTabClass: '',
    orders: [],
    loading: false,
    emptyVisible: false,
    operatingId: '',
    selfId: ''
  },

  onLoad(options) {
    const role = options.role === 'seller' ? 'seller' : 'buyer'
    this.setRole(role)
  },

  onShow() {
    if (!requireLogin(this.currentUrl(), { replace: true })) return
    const user = getUser() || {}
    this.setData({ selfId: user.id || '' })
    this.load()
  },

  setRole(role) {
    this.setData({
      role,
      buyerTabClass: role === 'buyer' ? 'active' : '',
      sellerTabClass: role === 'seller' ? 'active' : ''
    })
  },

  switchRole(e) {
    this.setRole(e.currentTarget.dataset.role)
    this.load()
  },

  async load() {
    this.setData({ loading: true })
    try {
      const data = await request({
        url: '/api/orders',
        auth: true,
        data: { role: this.data.role }
      })
      const orders = (data || []).map(item => ({
        ...item,
        coverUrl: assetUrl(item.productCover),
        priceText: formatPrice(item.productPrice),
        statusText: this.statusText(item.status),
        statusClass: this.statusClass(item.status),
        peerName: this.data.role === 'buyer' ? item.sellerName : item.buyerName,
        peerNameText: (this.data.role === 'buyer' ? item.sellerName : item.buyerName) || '同学',
        meetLocationText: item.meetLocation || '待沟通',
        canConfirm: this.data.role === 'seller' && item.status === 'PENDING',
        canFinish: item.status === 'RESERVED',
        canCancel: item.status === 'PENDING' || item.status === 'RESERVED'
      }))
      this.setData({
        orders,
        emptyVisible: orders.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  statusText(status) {
    const map = { PENDING: '待卖家确认', RESERVED: '已预约', COMPLETED: '已完成', CANCELLED: '已取消' }
    return map[status] || status
  },

  statusClass(status) {
    const map = { PENDING: 'pending', RESERVED: 'reserved', COMPLETED: 'done', CANCELLED: 'cancelled' }
    return map[status] || ''
  },

  operate(e) {
    const { id, action } = e.currentTarget.dataset
    const titleMap = { confirm: '确认预约', finish: '完成交易', cancel: '取消预约' }
    wx.showModal({
      title: titleMap[action] || '操作',
      content: '确认要执行这个操作吗？',
      success: async (res) => {
        if (!res.confirm) return
        this.setData({ operatingId: id })
        try {
          await request({
            url: `/api/orders/${id}/${action}`,
            method: 'PUT',
            auth: true
          })
          wx.showToast({ title: '已更新', icon: 'success' })
          this.load()
        } finally {
          this.setData({ operatingId: '' })
        }
      }
    })
  },

  contact(e) {
    const item = this.data.orders[e.currentTarget.dataset.index]
    if (!item) return
    const peerId = this.data.role === 'buyer' ? item.sellerId : item.buyerId
    wx.navigateTo({
      url: `/pages/chat/chat?peerId=${peerId}&peerName=${encodeURIComponent(item.peerName || '同学')}&productId=${item.productId}&productTitle=${encodeURIComponent(item.productTitle || '')}`
    })
  },

  goDetail(e) {
    wx.navigateTo({ url: `/pages/detail/detail?id=${e.currentTarget.dataset.id}` })
  },

  currentUrl() {
    return `/pages/orders/orders?role=${this.data.role}`
  }
})
