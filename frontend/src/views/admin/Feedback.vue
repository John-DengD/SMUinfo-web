<template>
  <div>
    <div class="toolbar">
      <h3>意见箱</h3>
      <el-radio-group v-model="status" size="small" @change="load">
        <el-radio-button label="">全部</el-radio-button>
        <el-radio-button label="PENDING">待处理</el-radio-button>
        <el-radio-button label="RESOLVED">已处理</el-radio-button>
        <el-radio-button label="CLOSED">已关闭</el-radio-button>
      </el-radio-group>
    </div>
    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="userName" label="用户" width="120">
        <template #default="{ row }">{{ row.userName || ('#' + (row.userId || '-')) }}</template>
      </el-table-column>
      <el-table-column prop="category" label="类型" width="110" />
      <el-table-column prop="content" label="内容" min-width="280" show-overflow-tooltip />
      <el-table-column prop="contact" label="联系方式" width="140" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="adminReply" label="回复" min-width="180" show-overflow-tooltip />
      <el-table-column prop="createdAt" label="提交时间" width="170">
        <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="openReply(row)">处理</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" title="处理意见" width="500px">
      <el-form :model="replyForm" label-width="80px">
        <el-form-item label="内容">
          <div class="content-preview">{{ replyForm.original }}</div>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="replyForm.status">
            <el-option label="待处理" value="PENDING" />
            <el-option label="已处理" value="RESOLVED" />
            <el-option label="已关闭" value="CLOSED" />
          </el-select>
        </el-form-item>
        <el-form-item label="回复">
          <el-input v-model="replyForm.adminReply" type="textarea" :rows="4" maxlength="500" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitReply">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { adminApi } from '../../api'

const list = ref([])
const loading = ref(false)
const status = ref('')
const dialogVisible = ref(false)
const replyForm = reactive({ id: null, status: 'RESOLVED', adminReply: '', original: '' })

const statusText = (s) => ({ PENDING: '待处理', RESOLVED: '已处理', CLOSED: '已关闭' }[s] || s)
const statusType = (s) => ({ PENDING: 'warning', RESOLVED: 'success', CLOSED: 'info' }[s] || '')
const formatTime = (t) => t ? new Date(t).toLocaleString() : '-'

const load = async () => {
  loading.value = true
  try {
    const { data } = await adminApi.feedback(status.value || undefined)
    list.value = data || []
  } finally { loading.value = false }
}

const openReply = (row) => {
  replyForm.id = row.id
  replyForm.status = row.status === 'PENDING' ? 'RESOLVED' : row.status
  replyForm.adminReply = row.adminReply || ''
  replyForm.original = row.content
  dialogVisible.value = true
}

const submitReply = async () => {
  await adminApi.replyFeedback(replyForm.id, { status: replyForm.status, adminReply: replyForm.adminReply })
  ElMessage.success('已保存')
  dialogVisible.value = false
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.toolbar h3 { margin: 0; }
.content-preview {
  background: #fafbff;
  border: 1px solid #eef0f7;
  border-radius: 6px;
  padding: 8px 10px;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 140px;
  overflow: auto;
  color: #303133;
}
</style>
