const assert = require('assert')
const path = require('path')

global.getApp = () => ({ globalData: {} })

function loadPageConfig(relativePath) {
  const pagePath = path.resolve(__dirname, relativePath)
  delete require.cache[pagePath]

  let config = null
  global.Page = pageConfig => {
    config = pageConfig
  }

  require(pagePath)

  delete global.Page
  return config
}

const pages = [
  '../pages/home/home.js',
  '../pages/detail/detail.js',
  '../pages/lost-found/lost-found.js',
  '../pages/lost-found-detail/lost-found-detail.js'
]

pages.forEach(page => {
  const config = loadPageConfig(page)
  assert.strictEqual(typeof config.onShareAppMessage, 'function', `${page} should define onShareAppMessage`)
  assert.strictEqual(typeof config.onShareTimeline, 'function', `${page} should define onShareTimeline`)
})

console.log('page share handler tests passed')
