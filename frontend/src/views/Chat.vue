<template>
  <div class="app-container">
    <div class="chat" v-loading="loading">
      <div class="chat-header">
        <span>与 {{ peerName }} 的对话</span>
        <el-button text @click="load">刷新</el-button>
      </div>
      <div class="chat-body" ref="bodyRef">
        <div v-for="m in messages" :key="m.id" class="msg" :class="{ self: m.senderId === userStore.user.id }">
          <div class="bubble">{{ m.content }}</div>
          <div class="time">{{ m.createdAt }}</div>
        </div>
        <div v-if="messages.length === 0" class="empty">还没有消息，发送第一条吧</div>
      </div>
      <div class="chat-input">
        <el-input v-model="content" type="textarea" :rows="2" placeholder="输入消息..." @keydown.enter.prevent="send" />
        <el-button type="primary" @click="send" :loading="sending">发送</el-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { messageApi } from '../api'
import { useUserStore } from '../stores/user'

const route = useRoute()
const userStore = useUserStore()
const peerId = computed(() => Number(route.params.userId))
const productId = computed(() => route.query.productId ? Number(route.query.productId) : null)
const messages = ref([])
const content = ref('')
const loading = ref(false)
const sending = ref(false)
const bodyRef = ref(null)
const peerName = ref('对方')

const load = async () => {
  loading.value = true
  try {
    const { data } = await messageApi.conversation(peerId.value)
    messages.value = data
    if (data.length) {
      const peer = data.find(m => m.senderId === peerId.value)
      if (peer) peerName.value = peer.senderName || peerName.value
    }
    await nextTick()
    if (bodyRef.value) bodyRef.value.scrollTop = bodyRef.value.scrollHeight
  } finally { loading.value = false }
}

const send = async () => {
  const text = content.value.trim()
  if (!text) return
  sending.value = true
  try {
    await messageApi.send({ receiverId: peerId.value, productId: productId.value, content: text })
    content.value = ''
    await load()
  } finally { sending.value = false }
}

onMounted(load)
</script>

<style scoped>
.chat {
  background: #fff;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 180px);
  overflow: hidden;
}
.chat-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  font-weight: 500;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.chat-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: #fafafa;
}
.msg { margin-bottom: 14px; display: flex; flex-direction: column; align-items: flex-start; }
.msg.self { align-items: flex-end; }
.bubble {
  background: #fff;
  padding: 8px 12px;
  border-radius: 8px;
  max-width: 70%;
  word-break: break-all;
}
.msg.self .bubble { background: var(--primary); color: #fff; }
.time { font-size: 11px; color: #909399; margin-top: 4px; }
.empty { text-align: center; color: #909399; padding: 24px 0; }
.chat-input {
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  display: flex;
  gap: 8px;
  align-items: flex-end;
}
</style>
