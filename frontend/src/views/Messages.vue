<template>
  <div class="app-container">
    <h2>消息中心</h2>
    <div v-if="conversations.length === 0 && !loading" class="empty-block">暂无消息</div>
    <div v-else class="conv-list" v-loading="loading">
      <div
        v-for="c in conversations"
        :key="c.peerId"
        class="conv-item"
        @click="$router.push(`/chat/${c.peerId}?productId=${c.productId || ''}`)"
      >
        <div class="avatar">{{ (c.peerName || '?').slice(0,1) }}</div>
        <div class="info">
          <div class="row">
            <span class="name">{{ c.peerName }}</span>
            <span class="time">{{ c.lastTime }}</span>
          </div>
          <div class="row">
            <span class="last">{{ c.lastContent }}</span>
            <el-badge v-if="c.unreadCount" :value="c.unreadCount" />
          </div>
          <div v-if="c.productTitle" class="product-ref">关于商品：{{ c.productTitle }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { messageApi } from '../api'

const conversations = ref([])
const loading = ref(false)
onMounted(async () => {
  loading.value = true
  try {
    const { data } = await messageApi.conversations()
    conversations.value = data
  } finally { loading.value = false }
})
</script>

<style scoped>
.conv-list { background: #fff; border-radius: 8px; overflow: hidden; }
.conv-item {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--border);
}
.conv-item:hover { background: #fafafa; }
.avatar {
  width: 44px; height: 44px; border-radius: 50%;
  background: var(--primary); color: #fff;
  display: flex; align-items: center; justify-content: center;
  font-weight: 600;
}
.info { flex: 1; }
.row { display: flex; justify-content: space-between; align-items: center; gap: 10px; }
.name { font-weight: 500; }
.time { font-size: 12px; color: #909399; }
.last { color: #606266; font-size: 13px; max-width: 70%; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.product-ref { font-size: 12px; color: #909399; margin-top: 4px; }
</style>
