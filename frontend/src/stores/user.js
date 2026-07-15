import { defineStore } from 'pinia'
import { identify } from '@hellyeah/x-ray'
import { authApi } from '../api'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    user: JSON.parse(localStorage.getItem('user') || 'null')
  }),
  getters: {
    isLoggedIn: (state) => !!state.token,
    isAdmin: (state) => state.user && state.user.role === 'ADMIN'
  },
  actions: {
    async login(form) {
      const { data } = await authApi.login(form)
      this.token = data.token
      this.user = data.user
      localStorage.setItem('token', data.token)
      localStorage.setItem('user', JSON.stringify(data.user))
      identify(String(data.user.id), { phone: data.user.phone || undefined })
      return data
    },
    async refreshMe() {
      try {
        const { data } = await authApi.me()
        this.user = data
        localStorage.setItem('user', JSON.stringify(data))
        identify(String(data.id), { phone: data.phone || undefined })
      } catch (e) {
        this.logout()
      }
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem('token')
      localStorage.removeItem('user')
    }
  }
})
