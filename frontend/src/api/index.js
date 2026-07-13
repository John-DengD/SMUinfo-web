import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'
import { useUserStore } from '../stores/user'

const api = axios.create({
  baseURL: '/api',
  timeout: 15000
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (resp) => {
    const data = resp.data
    if (data && typeof data.code !== 'undefined') {
      if (data.code === 0) return data
      if (data.code === 401) {
        handleUnauthorized()
        ElMessage.error(data.message || '登录已过期，请重新登录')
        return Promise.reject(new Error(data.message || '登录已过期，请重新登录'))
      }
      ElMessage.error(data.message || '请求失败')
      return Promise.reject(new Error(data.message || '请求失败'))
    }
    return resp
  },
  (err) => {
    if (err.response && err.response.status === 401) {
      handleUnauthorized()
      ElMessage.error(err.response.data?.message || '登录已过期，请重新登录')
    } else if (err.response && err.response.status === 403) {
      ElMessage.error(err.response.data?.message || '无权访问')
    } else {
      ElMessage.error(err.message || '网络错误')
    }
    return Promise.reject(err)
  }
)

function handleUnauthorized() {
  const userStore = useUserStore()
  userStore.logout()
  if (router.currentRoute.value.path !== '/login') {
    router.push({ path: '/login', query: { redirect: router.currentRoute.value.fullPath } })
  }
}

export default api

// 模块化封装
export const authApi = {
  register: (data) => api.post('/auth/register', data),
  login: (data) => api.post('/auth/login', data),
  me: () => api.get('/users/me'),
  updateMe: (data) => api.put('/users/me', data)
}
export const categoryApi = {
  list: () => api.get('/categories')
}
export const productApi = {
  list: (params) => api.get('/products', { params }),
  detail: (id) => api.get(`/products/${id}`),
  create: (data) => api.post('/products', data),
  update: (id, data) => api.put(`/products/${id}`, data),
  delete: (id) => api.delete(`/products/${id}`)
}
export const uploadApi = {
  image: (formData) => api.post('/upload/image', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}
export const favoriteApi = {
  add: (productId) => api.post(`/favorites/${productId}`),
  remove: (productId) => api.delete(`/favorites/${productId}`),
  list: () => api.get('/favorites')
}
export const messageApi = {
  send: (data) => api.post('/messages', data),
  conversations: () => api.get('/messages'),
  conversation: (userId) => api.get(`/messages/conversation/${userId}`),
  unreadCount: () => api.get('/messages/unread-count')
}
export const orderApi = {
  create: (data) => api.post('/orders', data),
  list: (role) => api.get('/orders', { params: { role } }),
  confirm: (id) => api.put(`/orders/${id}/confirm`),
  finish: (id) => api.put(`/orders/${id}/finish`),
  cancel: (id) => api.put(`/orders/${id}/cancel`)
}
export const reportApi = {
  create: (data) => api.post('/reports', data)
}
export const feedbackApi = {
  create: (data) => api.post('/feedback', data),
  mine: () => api.get('/feedback/mine')
}
export const adminApi = {
  users: (params) => api.get('/admin/users', { params }),
  userStatus: (id, status) => api.put(`/admin/users/${id}/status`, { status }),
  products: (params) => api.get('/admin/products', { params }),
  productStatus: (id, status) => api.put(`/admin/products/${id}/status`, { status }),
  categories: () => api.get('/admin/categories'),
  createCategory: (data) => api.post('/admin/categories', data),
  updateCategory: (id, data) => api.put(`/admin/categories/${id}`, data),
  deleteCategory: (id) => api.delete(`/admin/categories/${id}`),
  reports: (status) => api.get('/admin/reports', { params: { status } }),
  handleReport: (id, data) => api.put(`/admin/reports/${id}`, data),
  feedback: (status) => api.get('/admin/feedback', { params: { status } }),
  replyFeedback: (id, data) => api.put(`/admin/feedback/${id}`, data),
  announcements: () => api.get('/admin/announcements'),
  createAnnouncement: (data) => api.post('/admin/announcements', data),
  updateAnnouncement: (id, data) => api.put(`/admin/announcements/${id}`, data),
  deleteAnnouncement: (id) => api.delete(`/admin/announcements/${id}`)
}
export const announcementApi = {
  active: () => api.get('/announcements/active')
}
