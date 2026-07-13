const { request, getUser, clearSession } = require('../../utils/api')

Page({
  data: {
    user: null,
    initial: '?',
    collegeText: '学院未填写',
    campusText: '校区未填写',
    unreadCount: 0,
    messageSub: '查看买家和卖家的私信',
    isAdmin: false
  },

  onShow() {
    const user = getUser()
    this.setData({
      user,
      initial: user ? (user.name || '?').slice(0, 1) : '?',
      collegeText: user && user.college ? user.college : '学院未填写',
      campusText: user && user.campus ? user.campus : '校区未填写',
      isAdmin: user && user.role === 'ADMIN'
    })
    if (user) {
      this.refreshMe(true)
      this.loadUnread()
    } else {
      this.setData({ unreadCount: 0, messageSub: '查看买家和卖家的私信' })
    }
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login?redirect=%2Fpages%2Fme%2Fme' })
  },

  goFavorites() {
    wx.navigateTo({ url: '/pages/favorites/favorites' })
  },

  goMyProducts() {
    wx.navigateTo({ url: '/pages/my-products/my-products' })
  },

  goMessages() {
    wx.navigateTo({ url: '/pages/messages/messages' })
  },

  goOrders() {
    wx.navigateTo({ url: '/pages/orders/orders?role=seller' })
  },

  goProfile() {
    wx.navigateTo({ url: '/pages/profile/profile' })
  },

  goAdminUsers() {
    wx.navigateTo({ url: '/pages/admin-users/admin-users' })
  },

  goAdminProducts() {
    wx.navigateTo({ url: '/pages/admin-products/admin-products' })
  },

  goAdminAnnouncements() {
    wx.navigateTo({ url: '/pages/admin-announcements/admin-announcements' })
  },

  goHome() {
    wx.reLaunch({ url: '/pages/home/home' })
  },

  async loadUnread() {
    const data = await request({ url: '/api/messages/unread-count', auth: true, silent: true }).catch(() => ({ count: 0 }))
    const count = data.count || 0
    this.setData({
      unreadCount: count,
      messageSub: count ? `有 ${count} 条未读私信` : '查看买家和卖家的私信'
    })
  },

  async refreshMe(silent) {
    try {
      const user = await request({ url: '/api/users/me', auth: true, silent })
      wx.setStorageSync('user', user)
      this.setData({
        user,
        initial: (user.name || '?').slice(0, 1),
        collegeText: user.college || '学院未填写',
        campusText: user.campus || '校区未填写',
        isAdmin: user.role === 'ADMIN'
      })
      if (!silent) wx.showToast({ title: '已刷新', icon: 'success' })
    } catch (e) {
      if (!silent) wx.showToast({ title: '刷新失败', icon: 'none' })
    }
  },

  logout() {
    wx.showModal({
      title: '退出登录',
      content: '确认清除当前登录状态？',
      success: (res) => {
        if (!res.confirm) return
        clearSession()
        this.setData({
          user: null,
          initial: '?',
          collegeText: '学院未填写',
          campusText: '校区未填写',
          unreadCount: 0,
          messageSub: '查看买家和卖家的私信',
          isAdmin: false
        })
      }
    })
  }
})
