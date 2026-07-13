const assert = require('assert')
const {
  buildHomeShare,
  buildProductShare,
  buildLostFoundShare,
  buildLostFoundDetailShare
} = require('./share')

assert.deepStrictEqual(buildHomeShare('appMessage'), {
  title: 'Cute Jone 校园闲置',
  path: '/pages/home/home'
})

assert.deepStrictEqual(buildHomeShare('timeline'), {
  title: 'Cute Jone 校园闲置',
  query: ''
})

assert.deepStrictEqual(buildProductShare('appMessage', {
  id: 12,
  title: '高数教材',
  priceText: '18',
  descText: '几乎全新'
}, 'https://example.com/cover.jpg'), {
  title: '高数教材 · ¥18',
  path: '/pages/detail/detail?id=12',
  imageUrl: 'https://example.com/cover.jpg'
})

assert.deepStrictEqual(buildProductShare('timeline', {
  id: 12,
  title: '高数教材',
  priceText: '18'
}), {
  title: '高数教材 · ¥18',
  query: 'id=12'
})

assert.deepStrictEqual(buildLostFoundShare('appMessage'), {
  title: 'Cute Jone 失物招领',
  path: '/pages/lost-found/lost-found'
})

assert.deepStrictEqual(buildLostFoundDetailShare('timeline', {
  id: 8,
  typeText: '寻物',
  title: '校园卡'
}), {
  title: '寻物：校园卡',
  query: 'id=8'
})

console.log('share tests passed')
