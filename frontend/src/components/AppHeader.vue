<template>
  <header class="header">
    <div class="header-inner">
      <router-link to="/" class="logo">
        <span class="logo-mark">SMU</span>
        <span class="logo-text">
          <span class="logo-title">信息交易平台</span>
          <span class="logo-sub">校内闲置 · 信息撮合</span>
        </span>
      </router-link>
      <div class="search">
        <el-input
          v-model="keyword"
          placeholder="搜索教材 / 电子产品 / 宿舍用品..."
          clearable
          size="large"
          @keydown.enter="goSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
          <template #append>
            <el-button type="primary" @click="goSearch">搜索</el-button>
          </template>
        </el-input>
      </div>
      <nav class="nav">
        <router-link to="/" class="nav-link">首页</router-link>
        <router-link to="/publish" class="nav-link publish">
          <el-icon><Plus /></el-icon><span>发布闲置</span>
        </router-link>
        <router-link to="/messages" class="nav-link">
          <el-icon><ChatDotRound /></el-icon>
          <span>消息</span>
          <el-badge v-if="unread > 0" :value="unread" class="badge" />
        </router-link>
        <router-link to="/feedback" class="nav-link feedback-link">
          <el-icon><Promotion /></el-icon>
          <span>意见箱</span>
        </router-link>
        <template v-if="!userStore.isLoggedIn">
          <router-link to="/login" class="nav-link">登录</router-link>
          <router-link to="/register" class="nav-link register-link">注册</router-link>
        </template>
        <el-dropdown v-else trigger="click">
          <span class="nav-link user-chip">
            <span class="avatar">{{ (userStore.user?.name || '?').slice(0,1) }}</span>
            <span>{{ userStore.user.name }}</span>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="$router.push('/me')">我的资料</el-dropdown-item>
              <el-dropdown-item @click="$router.push('/my/products')">我的发布</el-dropdown-item>
              <el-dropdown-item @click="$router.push('/my/favorites')">我的收藏</el-dropdown-item>
              <el-dropdown-item @click="$router.push('/my/orders')">我的交易</el-dropdown-item>
              <el-dropdown-item @click="$router.push('/feedback')">意见箱</el-dropdown-item>
              <el-dropdown-item v-if="userStore.isAdmin" @click="$router.push('/admin')">管理后台</el-dropdown-item>
              <el-dropdown-item divided @click="onLogout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </nav>
    </div>
  </header>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search, Plus, ChatDotRound, Promotion } from '@element-plus/icons-vue'
import { track, cv } from '@hellyeah/x-ray'
import { useUserStore } from '../stores/user'
import { messageApi } from '../api'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const keyword = ref('')
const unread = ref(0)

const goSearch = () => {
  const q = (keyword.value || '').trim()
  if (q) track(cv.search, { query: q })
  router.push({ path: '/search', query: { keyword: keyword.value } })
}

const refreshUnread = async () => {
  if (!userStore.isLoggedIn) {
    unread.value = 0
    return
  }
  try {
    const { data } = await messageApi.unreadCount()
    unread.value = data.count
  } catch (e) { /* ignore */ }
}

const onLogout = () => {
  userStore.logout()
  router.push('/')
}

onMounted(refreshUnread)
watch(() => route.fullPath, refreshUnread)
watch(() => userStore.isLoggedIn, refreshUnread)
</script>

<style scoped>
.header {
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: saturate(180%) blur(14px);
  -webkit-backdrop-filter: saturate(180%) blur(14px);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 100;
}
.header-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 20px;
}
.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  white-space: nowrap;
}
.logo-mark {
  width: 48px;
  height: 38px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--primary), var(--accent));
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 14px;
  box-shadow: 0 6px 14px rgba(79, 70, 229, 0.3);
}
.logo-text {
  display: flex;
  flex-direction: column;
  line-height: 1.1;
}
.logo-title {
  font-size: 16px;
  font-weight: 700;
  color: #111827;
  letter-spacing: 0.2px;
}
.logo-sub {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
}
.search { flex: 1; max-width: 560px; }
.nav { display: flex; gap: 6px; align-items: center; }
.nav-link {
  color: #374151;
  text-decoration: none;
  font-size: 14px;
  padding: 8px 12px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: background .15s, color .15s;
}
.nav-link:hover {
  background: var(--primary-soft);
  color: var(--primary);
}
.router-link-active.nav-link:not(.publish):not(.register-link) {
  color: var(--primary);
  background: var(--primary-soft);
}
.nav-link.publish {
  background: linear-gradient(135deg, var(--primary), var(--accent));
  color: #fff !important;
  padding: 8px 14px;
  box-shadow: 0 6px 14px rgba(79, 70, 229, 0.28);
}
.nav-link.publish:hover { filter: brightness(1.05); }
.nav-link.register-link {
  border: 1px solid var(--primary);
  color: var(--primary);
}
.user-chip { padding: 6px 10px; }
.user-chip .avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary), var(--accent));
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
}
.badge { margin-left: 2px; }

@media (max-width: 900px) {
  .logo-sub { display: none; }
}
@media (max-width: 720px) {
  .header-inner { flex-wrap: wrap; gap: 10px; padding: 10px 12px; }
  .search { order: 3; flex-basis: 100%; }
  .logo { gap: 8px; }
  .logo-mark { width: 46px; height: 36px; font-size: 13px; }
  .logo-title { font-size: 15px; }
  .logo-sub { display: none; }
  .nav-link { padding: 6px 8px; font-size: 13px; }
  .nav-link.publish span { display: none; }
}
</style>
