<template>
  <div>
    <div class="toolbar">
      <h3>公告管理</h3>
      <el-button type="primary" @click="openCreate">发布公告</el-button>
    </div>

    <el-table :data="list" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="title" label="标题" width="180" show-overflow-tooltip />
      <el-table-column prop="content" label="内容" min-width="320" show-overflow-tooltip />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag size="small" :type="row.status === 'ACTIVE' ? 'success' : 'info'">
            {{ row.status === 'ACTIVE' ? '启用中' : '已停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="发布时间" width="180">
        <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">编辑</el-button>
          <el-button v-if="row.status !== 'ACTIVE'" size="small" type="success" @click="enable(row)">启用</el-button>
          <el-button size="small" type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑公告' : '发布公告'" width="560px">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="80px">
        <el-form-item label="标题" prop="title">
          <el-input v-model="form.title" maxlength="80" show-word-limit />
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input v-model="form.content" type="textarea" :rows="5" maxlength="500" show-word-limit />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio-button label="ACTIVE">启用</el-radio-button>
            <el-radio-button label="INACTIVE">停用</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '../../api'

const list = ref([])
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const formRef = ref(null)
const form = reactive({ id: null, title: '', content: '', status: 'ACTIVE' })

const rules = {
  title: [{ required: true, message: '请输入公告标题' }],
  content: [{ required: true, message: '请输入公告内容' }]
}

const formatTime = (t) => t ? new Date(t).toLocaleString() : '-'

const load = async () => {
  loading.value = true
  try {
    const { data } = await adminApi.announcements()
    list.value = data || []
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.id = null
  form.title = ''
  form.content = ''
  form.status = 'ACTIVE'
}

const openCreate = () => {
  resetForm()
  dialogVisible.value = true
}

const openEdit = (row) => {
  form.id = row.id
  form.title = row.title
  form.content = row.content
  form.status = row.status || 'INACTIVE'
  dialogVisible.value = true
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      const payload = { title: form.title, content: form.content, status: form.status }
      if (form.id) {
        await adminApi.updateAnnouncement(form.id, payload)
      } else {
        await adminApi.createAnnouncement(payload)
      }
      ElMessage.success('已保存')
      dialogVisible.value = false
      load()
    } finally {
      saving.value = false
    }
  })
}

const enable = async (row) => {
  await adminApi.updateAnnouncement(row.id, {
    title: row.title,
    content: row.content,
    status: 'ACTIVE'
  })
  ElMessage.success('已启用')
  load()
}

const remove = async (row) => {
  await ElMessageBox.confirm('确定删除这条公告吗？', '删除公告', { type: 'warning' })
  await adminApi.deleteAnnouncement(row.id)
  ElMessage.success('已删除')
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.toolbar h3 { margin: 0; }
</style>
