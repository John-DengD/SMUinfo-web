const { request, requireLogin, getUser } = require('../../utils/api')

Page({
  data: {
    peerId: '',
    peerName: '同学',
    peerAvatar: '同',
    productId: '',
    productTitle: '',
    productRefText: '普通私信',
    messages: [],
    content: '',
    loading: false,
    sending: false,
    scrollIntoView: ''
  },

  onLoad(options) {
    this.setData({
      peerId: options.peerId || '',
      peerName: options.peerName ? decodeURIComponent(options.peerName) : '同学',
      peerAvatar: (options.peerName ? decodeURIComponent(options.peerName) : '同学').slice(0, 1),
      productId: options.productId || '',
      productTitle: options.productTitle ? decodeURIComponent(options.productTitle) : '',
      productRefText: options.productTitle ? `关于：${decodeURIComponent(options.productTitle)}` : '普通私信'
    })
  },

  onShow() {
    if (!requireLogin(this.currentUrl(), { replace: true })) return
    this.load()
  },

  async load() {
    if (!this.data.peerId) return
    this.setData({ loading: true })
    try {
      const user = getUser() || {}
      const data = await request({ url: `/api/messages/conversation/${this.data.peerId}`, auth: true })
      const messages = (data || []).map(item => ({
        ...item,
        selfClass: item.senderId === user.id ? 'self' : '',
        timeText: this.formatTime(item.createdAt)
      }))
      this.setData({
        messages,
        scrollIntoView: messages.length ? `msg-${messages[messages.length - 1].id}` : ''
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  onInput(e) {
    this.setData({ content: e.detail.value })
  },

  async send() {
    const content = this.data.content.trim()
    if (!content) return
    if (content.length > 500) {
      wx.showToast({ title: '私信最多 500 字', icon: 'none' })
      return
    }
    this.setData({ sending: true })
    try {
      await request({
        url: '/api/messages',
        method: 'POST',
        auth: true,
        data: {
          receiverId: Number(this.data.peerId),
          productId: this.data.productId ? Number(this.data.productId) : null,
          content
        }
      })
      this.setData({ content: '' })
      await this.load()
    } finally {
      this.setData({ sending: false })
    }
  },

  currentUrl() {
    const query = [
      `peerId=${this.data.peerId}`,
      `peerName=${encodeURIComponent(this.data.peerName)}`
    ]
    if (this.data.productId) query.push(`productId=${this.data.productId}`)
    if (this.data.productTitle) query.push(`productTitle=${encodeURIComponent(this.data.productTitle)}`)
    return `/pages/chat/chat?${query.join('&')}`
  },

  formatTime(value) {
    if (!value) return ''
    const text = String(value).replace('T', ' ')
    return text.length > 16 ? text.slice(0, 16) : text
  }
})
