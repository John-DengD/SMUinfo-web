<template>
  <div class="app-container feedback-page">
    <div class="hero">
      <div class="hero-text">
        <h2>意见箱 / 给我们提建议</h2>
        <p>使用过程中遇到问题、想到新功能、或者只是想吐槽，都欢迎告诉我们。</p>
      </div>
      <div class="hero-emoji">💌</div>
    </div>

    <div class="grid">
      <div class="card form-card">
        <h3>我要提建议</h3>
        <el-form :model="form" :rules="rules" ref="formRef" label-position="top" @submit.prevent>
          <el-form-item label="意见类型" prop="category">
            <el-radio-group v-model="form.category">
              <el-radio-button label="功能建议">功能建议</el-radio-button>
              <el-radio-button label="Bug 反馈">Bug 反馈</el-radio-button>
              <el-radio-button label="体验问题">体验问题</el-radio-button>
              <el-radio-button label="其他">其他</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="详细描述" prop="content">
            <el-input
              v-model="form.content"
              type="textarea"
              :rows="6"
              maxlength="1000"
              show-word-limit
              placeholder="请尽量描述使用的页面、复现步骤、期望的效果……"
            />
          </el-form-item>
          <el-form-item label="联系方式（选填）">
            <el-input v-model="form.contact" placeholder="手机号 / 微信 / 学号，方便我们追问跟进" maxlength="64" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="loading" @click="submit" size="large">提交意见</el-button>
            <span class="hint">提交后管理员会在管理后台看到并跟进。</span>
          </el-form-item>
        </el-form>
      </div>

      <div class="card history-card">
        <div class="history-head">
          <h3>我提交的意见</h3>
          <el-button text @click="loadMine">刷新</el-button>
        </div>
        <div v-if="myList.length === 0" class="empty-block">还没有提交过意见</div>
        <div v-else class="history-list">
          <div v-for="item in myList" :key="item.id" class="history-item">
            <div class="row">
              <el-tag size="small" :type="tagType(item.category)">{{ item.category }}</el-tag>
              <el-tag size="small" :type="statusType(item.status)">{{ statusText(item.status) }}</el-tag>
              <span class="time">{{ formatTime(item.createdAt) }}</span>
            </div>
            <div class="content">{{ item.content }}</div>
            <div v-if="item.adminReply" class="reply">
              <strong>管理员回复：</strong>{{ item.adminReply }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { track } from '@hellyeah/x-ray'
import { feedbackApi } from '../api'

const formRef = ref(null)
const loading = ref(false)
const myList = ref([])
const form = reactive({ category: '功能建议', content: '', contact: '' })
const rules = {
  content: [{ required: true, message: '请描述你的意见' }, { min: 5, message: '至少 5 个字' }]
}

const tagType = (c) => ({
  '功能建议': 'success', 'Bug 反馈': 'danger', '体验问题': 'warning', '其他': 'info'
}[c] || '')

const statusText = (s) => ({ PENDING: '待处理', RESOLVED: '已处理', CLOSED: '已关闭' }[s] || s)
const statusType = (s) => ({ PENDING: 'warning', RESOLVED: 'success', CLOSED: 'info' }[s] || '')

const formatTime = (t) => t ? new Date(t).toLocaleString() : '-'

const loadMine = async () => {
  try {
    const { data } = await feedbackApi.mine()
    myList.value = data || []
  } catch (e) { /* api 已提示 */ }
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      await feedbackApi.create(form)
      track('feedback_submitted', { category: form.category })
      ElMessage.success('感谢你的反馈！我们已经收到啦')
      form.content = ''
      form.contact = ''
      loadMine()
    } catch (e) { /* ignore */ } finally { loading.value = false }
  })
}

onMounted(loadMine)
</script>

<style scoped>
.feedback-page { padding-top: 16px; padding-bottom: 32px; }
.hero {
  background: linear-gradient(135deg, #a78bfa 0%, #6366f1 60%, #06b6d4 100%);
  color: #fff;
  border-radius: 16px;
  padding: 28px 28px;
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 10px 30px rgba(99, 102, 241, 0.25);
}
.hero h2 { margin: 0 0 6px 0; font-size: 22px; }
.hero p { margin: 0; opacity: 0.92; }
.hero-emoji { font-size: 56px; line-height: 1; }
.grid {
  display: grid;
  grid-template-columns: 1.2fr 1fr;
  gap: 20px;
}
.card {
  background: #fff;
  border-radius: 14px;
  padding: 24px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.04);
}
.card h3 { margin: 0 0 16px 0; font-size: 16px; }
.form-card .hint { color: #909399; font-size: 12px; margin-left: 12px; }
.history-head { display: flex; align-items: center; justify-content: space-between; }
.history-list { display: flex; flex-direction: column; gap: 14px; max-height: 520px; overflow: auto; padding-right: 4px; }
.history-item {
  background: #fafbff;
  border: 1px solid #eef0f7;
  border-radius: 10px;
  padding: 12px 14px;
}
.history-item .row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.history-item .time { margin-left: auto; color: #909399; font-size: 12px; }
.history-item .content { color: #303133; line-height: 1.6; white-space: pre-wrap; word-break: break-word; }
.history-item .reply {
  margin-top: 8px;
  padding: 8px 10px;
  background: #f0f9ff;
  border-radius: 6px;
  color: #0369a1;
  font-size: 13px;
}
@media (max-width: 900px) {
  .grid { grid-template-columns: 1fr; }
  .hero { padding: 20px; }
  .hero-emoji { display: none; }
}
</style>
