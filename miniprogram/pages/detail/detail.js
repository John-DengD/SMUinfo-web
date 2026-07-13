const { request, assetUrl, formatPrice, requireLogin, getUser } = require('../../utils/api')
const { buildProductShare } = require('../../utils/share')

Page({
  data: {
    id: '',
    product: null,
    images: [],
    comments: [],
    commentContent: '',
    commentLoading: false,
    commentsEmpty: false,
    statusText: '',
    sellerInitial: '?',
    loading: false,
    wantLoading: false
  },

  onLoad(options) {
    this.setData({ id: options.id })
    this.load()
  },

  onShareAppMessage() {
    return buildProductShare('appMessage', this.shareProduct(), this.data.images[0])
  },

  onShareTimeline() {
    return buildProductShare('timeline', this.shareProduct(), this.data.images[0])
  },

  shareProduct() {
    return this.data.product || { id: this.data.id }
  },

  async load() {
    this.setData({ loading: true })
    try {
      const [product, comments] = await Promise.all([
        request({ url: `/api/products/${this.data.id}`, silent: true }),
        this.loadComments()
      ])
      const statusMap = { ON_SALE: '在售', RESERVED: '已预约', SOLD: '已售出', OFFLINE: '已下架' }
      this.setData({
        product: {
          ...product,
          priceText: formatPrice(product.price),
          conditionText: product.conditionLevel || '-',
          locationText: product.tradeLocation || '-',
          categoryText: product.categoryName || '-',
          viewText: product.viewCount || 0,
          descText: product.description || '卖家暂未填写描述',
          sellerNameText: product.sellerName || '同学',
          sellerCampusText: product.sellerCampus || '校园',
          favoriteText: product.favorited ? '已收藏' : '收藏',
          canReserve: product.status === 'ON_SALE'
        },
        images: (product.images || []).map(assetUrl),
        statusText: statusMap[product.status] || '',
        sellerInitial: (product.sellerName || '?').slice(0, 1)
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  async loadComments() {
    const comments = await request({
      url: `/api/products/${this.data.id}/comments`,
      silent: true
    }).catch(() => [])
    const list = (comments || []).map(item => ({
      ...item,
      avatarText: (item.userName || '?').slice(0, 1),
      displayName: this.commentUserText(item),
      timeText: this.formatTime(item.createdAt)
    }))
    this.setData({
      comments: list,
      commentsEmpty: list.length === 0
    })
    return list
  },

  async toggleFavorite() {
    if (!requireLogin(`/pages/detail/detail?id=${this.data.id}`)) return
    const product = this.data.product
    if (!product) return
    if (product.favorited) {
      await request({ url: `/api/favorites/${product.id}`, method: 'DELETE', auth: true })
      product.favorited = false
      product.favoriteText = '收藏'
      wx.showToast({ title: '已取消收藏', icon: 'none' })
    } else {
      await request({ url: `/api/favorites/${product.id}`, method: 'POST', auth: true })
      product.favorited = true
      product.favoriteText = '已收藏'
      wx.showToast({ title: '收藏成功', icon: 'success' })
    }
    this.setData({ product })
  },

  contactSeller() {
    if (!requireLogin(`/pages/detail/detail?id=${this.data.id}`)) return
    const user = getUser()
    const product = this.data.product
    if (user && product && user.id === product.sellerId) {
      wx.showToast({ title: '这是你发布的商品', icon: 'none' })
      return
    }
    wx.navigateTo({
      url: `/pages/chat/chat?peerId=${product.sellerId}&peerName=${encodeURIComponent(product.sellerNameText)}&productId=${product.id}&productTitle=${encodeURIComponent(product.title)}`
    })
  },

  reserveProduct() {
    if (!requireLogin(`/pages/detail/detail?id=${this.data.id}`)) return
    const user = getUser()
    const product = this.data.product
    if (!product) return
    if (user && user.id === product.sellerId) {
      wx.showToast({ title: '不能预约自己的商品', icon: 'none' })
      return
    }
    if (!product.canReserve) {
      wx.showToast({ title: '商品当前不可预约', icon: 'none' })
      return
    }
    wx.showModal({
      title: '提交预约申请',
      content: '提交后等待卖家确认，确认前商品仍可能被其他同学申请。',
      success: async (res) => {
        if (!res.confirm) return
        this.setData({ wantLoading: true })
        try {
          await request({
            url: '/api/orders',
            method: 'POST',
            auth: true,
            data: {
              productId: product.id,
              meetLocation: product.tradeLocation || '',
              remark: '我想要这个商品'
            }
          })
          wx.showToast({ title: '已提交给卖家', icon: 'success' })
          setTimeout(() => {
            wx.navigateTo({ url: '/pages/orders/orders?role=buyer' })
          }, 450)
        } finally {
          this.setData({ wantLoading: false })
        }
      }
    })
  },

  onCommentInput(e) {
    this.setData({ commentContent: e.detail.value })
  },

  async submitComment() {
    if (!requireLogin(`/pages/detail/detail?id=${this.data.id}`)) return
    const content = this.data.commentContent.trim()
    if (!content) {
      wx.showToast({ title: '请输入留言内容', icon: 'none' })
      return
    }
    if (content.length > 300) {
      wx.showToast({ title: '留言最多 300 字', icon: 'none' })
      return
    }
    this.setData({ commentLoading: true })
    try {
      const item = await request({
        url: `/api/products/${this.data.id}/comments`,
        method: 'POST',
        auth: true,
        data: { content }
      })
      const comment = {
        ...item,
        avatarText: (item.userName || '?').slice(0, 1),
        displayName: this.commentUserText(item),
        timeText: this.formatTime(item.createdAt)
      }
      this.setData({
        comments: this.data.comments.concat(comment),
        commentsEmpty: false,
        commentContent: ''
      })
      wx.showToast({ title: '留言成功', icon: 'success' })
    } finally {
      this.setData({ commentLoading: false })
    }
  },

  commentUserText(item) {
    const name = item.userName || '同学'
    return item.studentNoSuffix ? `${name} · ${item.studentNoSuffix}` : name
  },

  formatTime(value) {
    if (!value) return ''
    const text = String(value).replace('T', ' ')
    return text.length > 16 ? text.slice(0, 16) : text
  }
})
