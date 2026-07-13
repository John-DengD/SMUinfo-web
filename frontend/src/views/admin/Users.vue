<template>
  <div>
    <div class="bar">
      <el-input v-model="keyword" placeholder="搜索姓名/学号/手机" clearable style="width: 240px" @keydown.enter="load" />
      <el-button type="primary" @click="load">搜索</el-button>
    </div>
    <el-table :data="list" v-loading="loading" border>
      <el-table-column label="ID" prop="id" width="80" />
      <el-table-column label="姓名" prop="name" />
      <el-table-column label="学号" prop="studentNo" />
      <el-table-column label="手机" prop="phone" />
      <el-table-column label="学院" prop="college" />
      <el-table-column label="校区" prop="campus" />
      <el-table-column label="角色" prop="role" width="100" />
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'DISABLED' ? 'danger' : 'success'">
            {{ row.status === 'DISABLED' ? '已禁用' : '正常' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button v-if="row.status !== 'DISABLED'" size="small" type="danger" @click="setStatus(row, 'DISABLED')">禁用</el-button>
          <el-button v-else size="small" type="primary" @click="setStatus(row, 'ACTIVE')">启用</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="pagination">
      <el-pagination
        background
        layout="prev, pager, next"
        :total="total"
        :page-size="size"
        :current-page="page"
        @current-change="(p) => { page = p; load() }"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { adminApi } from '../../api'

const keyword = ref('')
const list = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)

const load = async () => {
  loading.value = true
  try {
    const { data } = await adminApi.users({ keyword: keyword.value, page: page.value, size: size.value })
    list.value = data.records
    total.value = data.total
    list.value.forEach(u => u.status = u.status || 'ACTIVE')
  } finally { loading.value = false }
}

const setStatus = async (row, status) => {
  await adminApi.userStatus(row.id, status)
  ElMessage.success('已更新')
  await load()
}

onMounted(load)
</script>

<style scoped>
.bar { display: flex; gap: 8px; margin-bottom: 12px; }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
