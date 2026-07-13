const { request, assetUrl, requireLogin } = require('../../utils/api')
const { buildLostFoundShare } = require('../../utils/share')

Page({
  data: {
    type: '',
    keyword: '',
    items: [],
    page: 1,
    size: 12,
    total: 0,
    loading: false,
    emptyVisible: false,
    finished: false,
    allClass: 'active',
    lostClass: '',
    foundClass: ''
  },

  onLoad() {
    this.load(true)
  },

  onShareAppMessage() {
    return buildLostFoundShare('appMessage')
  },

  onShareTimeline() {
    return buildLostFoundShare('timeline')
  },

  onPullDownRefresh() {
    this.load(true).finally(() => wx.stopPullDownRefresh())
  },

  onReachBottom() {
    if (!this.data.finished && !this.data.loading) this.load(false)
  },

  onKeywordInput(e) {
    this.setData({ keyword: e.detail.value })
  },

  onSearch() {
    this.load(true)
  },

  selectType(e) {
    const type = e.currentTarget.dataset.type || ''
    this.setData({
      type,
      allClass: type === '' ? 'active' : '',
      lostClass: type === 'LOST' ? 'active' : '',
      foundClass: type === 'FOUND' ? 'active' : ''
    })
    this.load(true)
  },

  async load(reset) {
    if (reset) this.setData({ page: 1, finished: false, emptyVisible: false })
    this.setData({ loading: true })
    try {
      const params = {
        page: this.data.page,
        size: this.data.size
      }
      if (this.data.type) params.type = this.data.type
      const keyword = this.data.keyword.trim()
      if (keyword) params.keyword = keyword
      const data = await request({ url: '/api/lost-found', data: params, silent: true })
      const records = (data.records || []).map(item => ({
        ...item,
        coverUrl: assetUrl(item.cover),
        typeClass: item.type === 'LOST' ? 'lost' : 'found',
        metaText: `${item.location || '地点未填写'} · ${this.formatTime(item.createdAt)}`,
        descText: item.description || ''
      }))
      const items = reset ? records : this.data.items.concat(records)
      this.setData({
        items,
        total: data.total || 0,
        page: this.data.page + 1,
        finished: items.length >= (data.total || 0),
        emptyVisible: items.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  goPublish() {
    if (!requireLogin('/pages/lost-found-publish/lost-found-publish')) return
    wx.navigateTo({ url: '/pages/lost-found-publish/lost-found-publish' })
  },

  goDetail(e) {
    wx.navigateTo({ url: `/pages/lost-found-detail/lost-found-detail?id=${e.currentTarget.dataset.id}` })
  },

  formatTime(value) {
    if (!value) return ''
    return String(value).replace('T', ' ').slice(0, 16)
  }
})
