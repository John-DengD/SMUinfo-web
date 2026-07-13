const HOME_TITLE = 'Cute Jone 校园闲置'
const LOST_FOUND_TITLE = 'Cute Jone 失物招领'

function encodeQueryValue(value) {
  if (value === undefined || value === null || value === '') return ''
  return encodeURIComponent(String(value))
}

function appendImage(config, imageUrl) {
  if (imageUrl) config.imageUrl = imageUrl
  return config
}

function buildByTarget(target, title, page, query, imageUrl) {
  if (target === 'timeline') {
    return appendImage({
      title,
      query: query || ''
    }, imageUrl)
  }

  return appendImage({
    title,
    path: query ? `${page}?${query}` : page
  }, imageUrl)
}

function productTitle(product) {
  const title = (product && product.title) || HOME_TITLE
  const priceText = product && product.priceText
  return priceText ? `${title} · ¥${priceText}` : title
}

function lostFoundDetailTitle(item) {
  const title = (item && item.title) || LOST_FOUND_TITLE
  const typeText = item && item.typeText
  return typeText ? `${typeText}：${title}` : title
}

function buildHomeShare(target) {
  return buildByTarget(target, HOME_TITLE, '/pages/home/home', '')
}

function buildProductShare(target, product, imageUrl) {
  const id = encodeQueryValue(product && product.id)
  const query = id ? `id=${id}` : ''
  return buildByTarget(target, productTitle(product), '/pages/detail/detail', query, imageUrl)
}

function buildLostFoundShare(target) {
  return buildByTarget(target, LOST_FOUND_TITLE, '/pages/lost-found/lost-found', '')
}

function buildLostFoundDetailShare(target, item, imageUrl) {
  const id = encodeQueryValue(item && item.id)
  const query = id ? `id=${id}` : ''
  return buildByTarget(target, lostFoundDetailTitle(item), '/pages/lost-found-detail/lost-found-detail', query, imageUrl)
}

module.exports = {
  buildHomeShare,
  buildProductShare,
  buildLostFoundShare,
  buildLostFoundDetailShare
}
