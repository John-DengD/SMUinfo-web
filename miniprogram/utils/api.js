const BASE_URL = 'https://dealinfor.drebel.top'

function getToken() {
  return wx.getStorageSync('token') || ''
}

function setSession(token, user) {
  wx.setStorageSync('token', token)
  wx.setStorageSync('user', user)
}

function clearSession() {
  wx.removeStorageSync('token')
  wx.removeStorageSync('user')
}

function getUser() {
  return wx.getStorageSync('user') || null
}

function normalizeUrl(url) {
  if (!url) return ''
  if (/^https?:\/\//.test(url)) return url
  if (url.startsWith('/api/')) return BASE_URL + url
  if (url.startsWith('/')) return BASE_URL + url
  return BASE_URL + '/api/' + url
}

function assetUrl(url) {
  return normalizeUrl(url)
}

function cleanData(data) {
  if (!data || Array.isArray(data) || typeof data !== 'object') return data
  const cleaned = {}
  Object.keys(data).forEach(key => {
    const value = data[key]
    if (value === undefined || value === null || value === '') return
    if (value === 'undefined' || value === 'null') return
    cleaned[key] = value
  })
  return cleaned
}

function currentPageUrl() {
  const pages = getCurrentPages()
  const page = pages[pages.length - 1]
  if (!page || !page.route) return ''
  const options = page.options || {}
  const query = Object.keys(options)
    .map(key => `${encodeURIComponent(key)}=${encodeURIComponent(options[key])}`)
    .join('&')
  return `/${page.route}${query ? `?${query}` : ''}`
}

function handleUnauthorized(message) {
  clearSession()
  if (message) wx.showToast({ title: message, icon: 'none' })
  const redirect = currentPageUrl()
  if (redirect.startsWith('/pages/login/login')) return
  wx.navigateTo({ url: loginUrl(redirect) })
}

function request(options) {
  const { url, method = 'GET', data = {}, auth = false, silent = false } = options
  const header = {
    'Content-Type': 'application/json'
  }
  const token = getToken()
  if (auth && token) {
    header.Authorization = `Bearer ${token}`
  }

  return new Promise((resolve, reject) => {
    wx.request({
      url: normalizeUrl(url),
      method,
      data: cleanData(data),
      header,
      success(res) {
        const body = res.data
        if (res.statusCode < 200 || res.statusCode >= 300) {
          if (res.statusCode === 401) {
            handleUnauthorized((body && body.message) || '登录已过期，请重新登录')
            reject(new Error('登录已过期，请重新登录'))
            return
          }
          const msg = (body && body.message) || (res.statusCode === 403 ? '无权访问' : `请求失败(${res.statusCode})`)
          if (!silent) wx.showToast({ title: msg, icon: 'none' })
          reject(new Error(msg))
          return
        }
        if (body && typeof body.code !== 'undefined') {
          if (body.code === 0) {
            resolve(body.data)
            return
          }
          if (body.code === 401) {
            handleUnauthorized(body.message || '登录已过期，请重新登录')
            reject(new Error(body.message || '登录已过期，请重新登录'))
            return
          }
          const msg = body.message || '请求失败'
          if (!silent) wx.showToast({ title: msg, icon: 'none' })
          reject(new Error(msg))
          return
        }
        resolve(body)
      },
      fail(err) {
        const msg = err.errMsg || '网络错误'
        if (!silent) wx.showToast({ title: msg, icon: 'none' })
        reject(new Error(msg))
      }
    })
  })
}

function uploadImage(filePath) {
  const token = getToken()
  return new Promise((resolve, reject) => {
    wx.uploadFile({
      url: BASE_URL + '/api/upload/image',
      filePath,
      name: 'file',
      header: token ? { Authorization: `Bearer ${token}` } : {},
      success(res) {
        let body = {}
        try {
          body = JSON.parse(res.data)
        } catch (e) {
          wx.showToast({ title: '上传返回格式错误', icon: 'none' })
          reject(e)
          return
        }
        if (res.statusCode < 200 || res.statusCode >= 300) {
          if (res.statusCode === 401) {
            handleUnauthorized(body.message || '登录已过期，请重新登录')
            reject(new Error('登录已过期，请重新登录'))
            return
          }
          const msg = body.message || (res.statusCode === 403 ? '无权访问' : '上传失败')
          wx.showToast({ title: msg, icon: 'none' })
          reject(new Error(msg))
          return
        }
        if (body.code === 0) {
          resolve(body.data)
          return
        }
        if (body.code === 401) {
          handleUnauthorized(body.message || '登录已过期，请重新登录')
          reject(new Error(body.message || '登录已过期，请重新登录'))
          return
        }
        const msg = body.message || '上传失败'
        wx.showToast({ title: msg, icon: 'none' })
        reject(new Error(msg))
      },
      fail(err) {
        wx.showToast({ title: err.errMsg || '上传失败', icon: 'none' })
        reject(err)
      }
    })
  })
}

function loginUrl(redirect) {
  const next = redirect ? `?redirect=${encodeURIComponent(redirect)}` : ''
  return `/pages/login/login${next}`
}

function requireLogin(redirect, options = {}) {
  if (getToken()) return true
  const url = loginUrl(redirect)
  if (options.replace) {
    wx.redirectTo({ url })
  } else {
    wx.navigateTo({ url })
  }
  return false
}

function formatPrice(value) {
  if (value === null || value === undefined || value === '') return '0'
  const n = Number(value)
  if (Number.isNaN(n)) return String(value)
  return n % 1 === 0 ? String(n) : n.toFixed(2)
}

module.exports = {
  BASE_URL,
  request,
  uploadImage,
  assetUrl,
  formatPrice,
  loginUrl,
  getToken,
  getUser,
  setSession,
  clearSession,
  requireLogin
}
