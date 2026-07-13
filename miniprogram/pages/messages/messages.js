const { request, requireLogin } = require('../../utils/api')

Page({
  data: {
    conversations: [],
    loading: false,
    emptyVisible: false
  },

  onShow() {
    if (!requireLogin('/pages/messages/messages', { replace: true })) return
    this.load()
  },

  async load() {
    this.setData({ loading: true })
    try {
      const data = await request({ url: '/api/messages', auth: true })
      const conversations = (data || []).map(item => ({
        ...item,
        peerNameText: item.peerName || '同学',
        avatarText: (item.peerName || '?').slice(0, 1),
        lastContentText: item.lastContent || '暂无消息',
        timeText: this.formatTime(item.lastTime),
        productText: item.productTitle ? `关于：${item.productTitle}` : '普通私信'
      }))
      this.setData({
        conversations,
        emptyVisible: conversations.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  goChat(e) {
    const item = this.data.conversations[e.currentTarget.dataset.index]
    if (!item) return
    const query = [
      `peerId=${item.peerId}`,
      `peerName=${encodeURIComponent(item.peerNameText)}`
    ]
    if (item.productId) query.push(`productId=${item.productId}`)
    if (item.productTitle) query.push(`productTitle=${encodeURIComponent(item.productTitle)}`)
    wx.navigateTo({ url: `/pages/chat/chat?${query.join('&')}` })
  },

  formatTime(value) {
    if (!value) return ''
    const text = String(value).replace('T', ' ')
    return text.length > 16 ? text.slice(0, 16) : text
  }
})
