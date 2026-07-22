<template>
  <div class="layout">
    <Analytics :website-id="TRACKER_ID" :env="trackerEnv" />
    <AppHeader />
    <main class="layout-main">
      <router-view />
    </main>
    <footer class="layout-footer">
      <div class="footer-inner">
        <span class="brand">SMU 信息交易平台</span>
        <span class="dot">·</span>
        <span>仅信息撮合，所有交易请校内当面进行</span>
        <span class="dot">·</span>
        <router-link to="/feedback" class="footer-link">意见箱</router-link>
      </div>
    </footer>
  </div>
</template>

<script setup>
import AppHeader from './components/AppHeader.vue'
import { Analytics } from '@hellyeah/x-ray/vue'
import { onMounted } from 'vue'
import { useUserStore } from './stores/user'
import { TRACKER_ID } from './tracker'

const trackerEnv = import.meta.env.VITE_TRACKER_ENV
const userStore = useUserStore()
onMounted(() => {
  if (userStore.isLoggedIn) {
    userStore.refreshMe()
  }
})
</script>

<style scoped>
.layout {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-page);
}
.layout-main {
  flex: 1;
}
.layout-footer {
  margin-top: 32px;
  padding: 20px 16px;
  border-top: 1px solid var(--border);
  background: #fff;
}
.footer-inner {
  max-width: 1200px;
  margin: 0 auto;
  text-align: center;
  font-size: 12px;
  color: var(--text-muted);
  display: flex;
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.footer-inner .brand { color: var(--primary); font-weight: 600; }
.footer-inner .dot { color: #d1d5db; }
.footer-link { color: var(--text-muted); text-decoration: none; }
.footer-link:hover { color: var(--primary); }
</style>
