import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const routes = [
  { path: '/', component: () => import('../views/Home.vue') },
  { path: '/search', component: () => import('../views/Search.vue') },
  { path: '/product/:id', component: () => import('../views/ProductDetail.vue') },
  { path: '/login', component: () => import('../views/Login.vue') },
  { path: '/register', component: () => import('../views/Register.vue') },
  { path: '/publish', component: () => import('../views/Publish.vue'), meta: { requireAuth: true } },
  { path: '/publish/:id', component: () => import('../views/Publish.vue'), meta: { requireAuth: true } },
  { path: '/me', component: () => import('../views/Me.vue'), meta: { requireAuth: true } },
  { path: '/my/products', component: () => import('../views/MyProducts.vue'), meta: { requireAuth: true } },
  { path: '/my/favorites', component: () => import('../views/MyFavorites.vue'), meta: { requireAuth: true } },
  { path: '/my/orders', component: () => import('../views/MyOrders.vue'), meta: { requireAuth: true } },
  { path: '/messages', component: () => import('../views/Messages.vue'), meta: { requireAuth: true } },
  { path: '/chat/:userId', component: () => import('../views/Chat.vue'), meta: { requireAuth: true } },
  { path: '/feedback', component: () => import('../views/Feedback.vue'), meta: { requireAuth: true } },
  {
    path: '/admin',
    component: () => import('../views/admin/AdminLayout.vue'),
    meta: { requireAuth: true, requireAdmin: true },
    children: [
      { path: '', redirect: '/admin/users' },
      { path: 'users', component: () => import('../views/admin/Users.vue') },
      { path: 'products', component: () => import('../views/admin/Products.vue') },
      { path: 'categories', component: () => import('../views/admin/Categories.vue') },
      { path: 'reports', component: () => import('../views/admin/Reports.vue') },
      { path: 'feedback', component: () => import('../views/admin/Feedback.vue') },
      { path: 'announcements', component: () => import('../views/admin/Announcements.vue') }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() { return { top: 0 } }
})

router.beforeEach((to, from, next) => {
  const userStore = useUserStore()
  if (to.meta.requireAuth && !userStore.isLoggedIn) {
    return next({ path: '/login', query: { redirect: to.fullPath } })
  }
  if (to.meta.requireAdmin && !userStore.isAdmin) {
    return next('/')
  }
  next()
})

export default router
