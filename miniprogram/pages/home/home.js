const { request, assetUrl, formatPrice, requireLogin } = require('../../utils/api')
const { buildHomeShare } = require('../../utils/share')
const app = getApp()

const TRANSIT_ROUTES = [
  {
    label: '龙阳路到临港大道',
    line: 'METRO_16',
    station: '龙阳路',
    direction: 'TO_DISHUI'
  },
  {
    label: '临港大道到龙阳路',
    line: 'METRO_16',
    station: '临港大道',
    direction: 'TO_LONGYANG'
  },
  {
    label: '罗山路到临港大道',
    line: 'METRO_16',
    station: '罗山路',
    direction: 'TO_DISHUI'
  },
  {
    label: '共享区到临港大道',
    line: 'BUS_1077',
    station: '临港共享区枢纽站',
    direction: 'TO_LINGANG_AVE'
  },
  {
    label: '临港大道到共享区',
    line: 'BUS_1077',
    station: '临港大道枢纽站',
    direction: 'TO_SHARED'
  }
]

Page({
  data: {
    categories: [],
    products: [],
    keyword: '',
    currentCategory: '',
    currentCategoryName: '全部',
    categoryToggleText: '全部 展开',
    allCategoryActive: 'active',
    categoryExpanded: false,
    sortBy: '',
    sortLatestClass: 'active',
    sortPriceAscClass: '',
    sortPriceDescClass: '',
    sortHotClass: '',
    page: 1,
    size: 12,
    total: 0,
    loading: false,
    emptyVisible: false,
    finished: false,
    transitLoading: false,
    transitLine: 'METRO_16',
    transitStation: '龙阳路',
    transitDirection: 'TO_DISHUI',
    transitRoutes: TRANSIT_ROUTES,
    transitRouteLabels: TRANSIT_ROUTES.map(item => item.label),
    transitRouteIndex: 0,
    transitRouteLabel: TRANSIT_ROUTES[0].label,
    transitRouteText: TRANSIT_ROUTES[0].label,
    transitScheduleText: '正在读取时刻',
    transitNearestText: '--',
    transitNextText: '--',
    transitSpecialText: '',
    transitError: '',
    announcement: null,
    announcementVisible: false
  },

  onLoad() {
    this.syncTransitRoute(0)
    this.loadCategories()
    this.loadAnnouncement()
    this.loadProducts(true)
    this.loadTransit(true)
    this.startTransitTimer()
  },

  onShareAppMessage() {
    return buildHomeShare('appMessage')
  },

  onShareTimeline() {
    return buildHomeShare('timeline')
  },

  onShow() {
    if (this.data.products.length) {
      this.loadProducts(true)
    }
    this.loadAnnouncement()
    this.loadTransit(true)
    this.startTransitTimer()
  },

  onHide() {
    this.stopTransitTimer()
  },

  onUnload() {
    this.stopTransitTimer()
  },

  onPullDownRefresh() {
    Promise.all([this.loadCategories(), this.loadAnnouncement(), this.loadProducts(true), this.loadTransit(true)]).finally(() => {
      wx.stopPullDownRefresh()
    })
  },

  onReachBottom() {
    if (!this.data.finished && !this.data.loading) {
      this.loadProducts(false)
    }
  },

  async loadCategories() {
    const categories = await request({ url: '/api/categories', silent: true }).catch(() => [])
    this.setData({ categories: this.decorateCategories(categories) })
  },

  async loadAnnouncement() {
    const announcement = await request({ url: '/api/announcements/active', silent: true }).catch(() => null)
    const dismissedIds = app.globalData.dismissedAnnouncementIds || []
    const visible = !!announcement && !dismissedIds.includes(announcement.id)
    this.setData({
      announcement,
      announcementVisible: visible
    })
  },

  closeAnnouncement() {
    const announcement = this.data.announcement
    if (announcement && announcement.id) {
      const dismissedIds = app.globalData.dismissedAnnouncementIds || []
      if (!dismissedIds.includes(announcement.id)) {
        dismissedIds.push(announcement.id)
      }
      app.globalData.dismissedAnnouncementIds = dismissedIds
    }
    this.setData({ announcementVisible: false })
  },

  async loadProducts(reset) {
    if (reset) {
      this.setData({ page: 1, finished: false, emptyVisible: false })
    }
    this.setData({ loading: true })
    try {
      const params = {
        page: this.data.page,
        size: this.data.size
      }
      const keyword = this.data.keyword.trim()
      const categoryId = this.normalizeCategoryId(this.data.currentCategory)
      if (keyword) params.keyword = keyword
      if (categoryId) params.categoryId = categoryId
      if (this.data.sortBy) params.sortBy = this.data.sortBy

      const data = await request({
        url: '/api/products',
        data: params
      })
      const records = (data.records || []).map(item => ({
        ...item,
        coverUrl: assetUrl(item.cover),
        priceText: formatPrice(item.price),
        metaText: `${item.conditionLevel || '九成新'} · ${item.tradeLocation || item.sellerCampus || '校园'}`
      }))
      const products = reset ? records : this.data.products.concat(records)
      this.setData({
        products,
        total: data.total || 0,
        page: this.data.page + 1,
        finished: products.length >= (data.total || 0),
        emptyVisible: products.length === 0
      })
    } finally {
      this.setData({ loading: false })
    }
  },

  onKeywordInput(e) {
    this.setData({ keyword: e.detail.value })
  },

  onSearch() {
    this.loadProducts(true)
  },

  toggleCategories() {
    const categoryExpanded = !this.data.categoryExpanded
    this.setData({
      categoryExpanded,
      categoryToggleText: `${this.data.currentCategoryName} ${categoryExpanded ? '收起' : '展开'}`
    })
  },

  selectCategory(e) {
    const currentCategory = this.normalizeCategoryId(e.currentTarget.dataset.id)
    const currentCategoryName = e.currentTarget.dataset.name || '全部'
    this.setData({
      currentCategory,
      currentCategoryName,
      categoryToggleText: `${currentCategoryName} 展开`,
      allCategoryActive: currentCategory ? '' : 'active',
      categories: this.decorateCategories(this.data.categories, currentCategory),
      categoryExpanded: false
    })
    this.loadProducts(true)
  },

  selectSort(e) {
    const sortBy = e.currentTarget.dataset.value || ''
    this.setData({
      sortBy,
      sortLatestClass: sortBy === '' ? 'active' : '',
      sortPriceAscClass: sortBy === 'price_asc' ? 'active' : '',
      sortPriceDescClass: sortBy === 'price_desc' ? 'active' : '',
      sortHotClass: sortBy === 'hot' ? 'active' : ''
    })
    this.loadProducts(true)
  },

  goDetail(e) {
    wx.navigateTo({ url: `/pages/detail/detail?id=${e.currentTarget.dataset.id}` })
  },

  goPublish() {
    if (!requireLogin('/pages/publish/publish')) return
    wx.navigateTo({ url: '/pages/publish/publish' })
  },

  goLostFound() {
    wx.navigateTo({ url: '/pages/lost-found/lost-found' })
  },

  goMe() {
    wx.reLaunch({ url: '/pages/me/me' })
  },

  refreshTransit() {
    this.loadTransit(false)
  },

  onTransitRouteChange(e) {
    this.syncTransitRoute(Number(e.detail.value) || 0)
    this.loadTransit(true)
  },

  async loadTransit(silent) {
    this.setData({ transitLoading: true })
    try {
      const data = await request({
        url: '/api/transit/next',
        data: {
          line: this.data.transitLine,
          station: this.data.transitStation,
          direction: this.data.transitDirection
        },
        silent: true
      })
      const nearestText = this.formatDeparture(data.nearest)
      const nextText = this.formatDeparture(data.next)
      const specialText = this.formatSpecial(data.specialNearest, data.specialNext, [data.nearest, data.next])
      this.setData({
        transitLoading: false,
        transitError: '',
        transitLine: data.line || this.data.transitLine,
        transitStation: data.station || this.data.transitStation,
        transitDirection: data.direction || this.data.transitDirection,
        transitRouteText: this.data.transitRouteLabel,
        transitScheduleText: data.scheduleType || '',
        transitNearestText: nearestText,
        transitNextText: nextText,
        transitSpecialText: specialText
      })
    } catch (err) {
      this.setData({
        transitLoading: false,
        transitError: '时刻暂时不可用',
        transitScheduleText: '时刻暂时不可用',
        transitNearestText: '--',
        transitNextText: '--',
        transitSpecialText: ''
      })
      if (!silent) wx.showToast({ title: '时刻暂时不可用', icon: 'none' })
    }
  },

  startTransitTimer() {
    this.stopTransitTimer()
    this.transitTimer = setInterval(() => {
      this.loadTransit(true)
    }, 60000)
  },

  stopTransitTimer() {
    if (!this.transitTimer) return
    clearInterval(this.transitTimer)
    this.transitTimer = null
  },

  syncTransitRoute(index) {
    const routeIndex = Math.min(Math.max(index, 0), TRANSIT_ROUTES.length - 1)
    const route = TRANSIT_ROUTES[routeIndex]
    this.setData({
      transitRouteIndex: routeIndex,
      transitRouteLabel: route.label,
      transitRouteText: route.label,
      transitLine: route.line,
      transitStation: route.station,
      transitDirection: route.direction
    })
  },

  formatDeparture(departure) {
    if (!departure || !departure.time) return '--'
    const label = departure.serviceLabel && departure.serviceLabel !== '普通车' ? departure.serviceLabel : ''
    return label ? `${departure.time} ${label}` : departure.time
  },

  formatSpecial(nearest, next, visibleDepartures) {
    const visibleSpecialTimes = (visibleDepartures || [])
      .filter(item => item && item.serviceType && item.serviceType !== 'NORMAL')
      .map(item => item.time)
    const items = [nearest, next]
      .filter(item => item && item.time && !visibleSpecialTimes.includes(item.time))
    if (!items.length) return ''
    const label = items[0].serviceLabel || '特殊车次'
    return `${label} ${items.map(item => item.time).join(' / ')}`
  },

  normalizeCategoryId(value) {
    if (value === null || value === undefined || value === '') return ''
    const text = String(value).trim()
    return /^\d+$/.test(text) ? text : ''
  },

  decorateCategories(categories, currentCategory) {
    const selected = this.normalizeCategoryId(currentCategory || this.data.currentCategory)
    return (categories || []).map(item => ({
      ...item,
      activeClass: String(selected) === String(item.id) ? 'active' : ''
    }))
  }
})
