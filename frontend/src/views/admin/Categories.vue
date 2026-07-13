<template>
  <div>
    <div class="bar">
      <el-button type="primary" @click="openDialog()">+ 新增分类</el-button>
    </div>
    <el-table :data="list" v-loading="loading" border>
      <el-table-column label="ID" prop="id" width="80" />
      <el-table-column label="名称" prop="name" />
      <el-table-column label="排序" prop="sortOrder" width="100" />
      <el-table-column label="状态" prop="status" width="100" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="openDialog(row)">编辑</el-button>
          <el-popconfirm title="确认删除？" @confirm="remove(row)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog" :title="form.id ? '编辑分类' : '新增分类'" width="400px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sortOrder" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status">
            <el-option label="启用" value="ACTIVE" />
            <el-option label="禁用" value="DISABLED" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
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
const dialog = ref(false)
const form = reactive({ id: null, name: '', sortOrder: 0, status: 'ACTIVE' })

const load = async () => {
  loading.value = true
  try {
    const { data } = await adminApi.categories()
    list.value = data
  } finally { loading.value = false }
}

const openDialog = (row) => {
  if (row) {
    form.id = row.id
    form.name = row.name
    form.sortOrder = row.sortOrder
    form.status = row.status
  } else {
    form.id = null
    form.name = ''
    form.sortOrder = 0
    form.status = 'ACTIVE'
  }
  dialog.value = true
}

const save = async () => {
  if (form.id) {
    await adminApi.updateCategory(form.id, form)
  } else {
    await adminApi.createCategory(form)
  }
  ElMessage.success('保存成功')
  dialog.value = false
  await load()
}

const remove = async (row) => {
  await adminApi.deleteCategory(row.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(load)
</script>

<style scoped>
.bar { display: flex; gap: 8px; margin-bottom: 12px; }
</style>
