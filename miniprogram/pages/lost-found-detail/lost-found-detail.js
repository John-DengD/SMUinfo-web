const { request, assetUrl, requireLogin, getUser } = require('../../utils/api')
const { buildLostFoundDetailShare } = require('../../utils/share')

Page({
  data: {
    id: '',
    item: null,
    images: [],
    loading: false,
    operating: false,
    userInitial: '?',
    canClose: false
  },

  onLoad(options) {
    this.setData({ id: options.id })
    this.load()
  },

  onShareAppMessage() {
    return buildLostFoundDetailShare('appMessage', this.shareItem(), this.data.images[0])
  },

  onShareTimeline() {
    return buildLostFoundDetailShare('timeline', this.shareItem(), this.data.images[0])
  },

  shareItem() {
    return this.data.item || { id: this.data.id }
  },

  async load() {
    this.setData({ loading: true })
    try {
      const item = await request({ url: `/api/lost-found/${this.data.id}`, silent: true })
      const user = getUser()
      this.setData({
        item: {
          ...item,
          descText: item.description || '',
          locationText: item.location || '地点未填写',
          contactText: item.contact || '可通过私信联系',
          timeText: this.formatTime(item.createdAt),
          viewText: item.viewCount || 0,
          userNameText: item.userName || '同学',
          userCampusText: item.userCampus || '校园',
          typeClass: item.type === 'LOST' ? 'lost' : 'found'
        },
        images: (item.images || []).map(assetUrl),
        userInitial: (item.userName || '?').slice(0, 1),
        canClose: !!user && (user.role === 'ADMIN' || user.id === item.userId)
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  contactPublisher() {
    if (!requireLogin(`/pages/lost-found-detail/lost-found-detail?id=${this.data.id}`)) return
    const user = getUser()
    const item = this.data.item
    if (!item) return
    if (user && user.id === item.userId) {
      wx.showToast({ title: '这是你发布的信息', icon: 'none' })
      return
    }
    wx.navigateTo({
      url: `/pages/chat/chat?peerId=${item.userId}&peerName=${encodeURIComponent(item.userNameText)}&productTitle=${encodeURIComponent(item.title)}`
    })
  },

  closeItem() {
    if (!requireLogin(`/pages/lost-found-detail/lost-found-detail?id=${this.data.id}`)) return
    wx.showModal({
      title: '关闭信息',
      content: '关闭后列表里不再展示这条失物招领信息。',
      success: async (res) => {
        if (!res.confirm) return
        this.setData({ operating: true })
        try {
          await request({ url: `/api/lost-found/${this.data.id}`, method: 'DELETE', auth: true })
          wx.showToast({ title: '已关闭', icon: 'success' })
          setTimeout(() => wx.navigateBack(), 450)
        } finally {
          this.setData({ operating: false })
        }
      }
    })
  },

  formatTime(value) {
    if (!value) return ''
    return String(value).replace('T', ' ').slice(0, 16)
  }
})
