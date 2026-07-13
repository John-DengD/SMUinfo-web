<template>
  <div>
    <div class="bar">
      <el-select v-model="status" placeholder="状态" clearable style="width: 160px" @change="load">
        <el-option label="待处理" value="PENDING" />
        <el-option label="已处理" value="RESOLVED" />
        <el-option label="已驳回" value="REJECTED" />
      </el-select>
    </div>
    <el-table :data="list" v-loading="loading" border>
      <el-table-column label="ID" prop="id" width="80" />
      <el-table-column label="商品">
        <template #default="{ row }">
          <router-link :to="`/product/${row.productId}`">{{ row.productTitle }}</router-link>
        </template>
      </el-table-column>
      <el-table-column label="举报人" prop="reporterName" width="120" />
      <el-table-column label="原因" prop="reason" />
      <el-table-column label="状态" width="120">
        <template #default="{ row }">{{ statusText(row.status) }}</template>
      </el-table-column>
      <el-table-column label="管理备注" prop="adminRemark" width="180" />
      <el-table-column label="时间" prop="createdAt" width="170" />
      <el-table-column label="操作" width="240">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="handle(row, 'RESOLVED')">下架并处理</el-button>
          <el-button size="small" @click="handle(row, 'REJECTED')">驳回</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '../../api'

const status = ref('PENDING')
const list = ref([])
const loading = ref(false)

const statusText = (s) => ({ PENDING: '待处理', RESOLVED: '已处理', REJECTED: '已驳回' }[s] || s)

const load = async () => {
  loading.value = true
  try {
    const { data } = await adminApi.reports(status.value)
    list.value = data
  } finally { loading.value = false }
}

const handle = async (row, newStatus) => {
  const { value: remark } = await ElMessageBox.prompt('管理备注', '处理举报', {
    confirmButtonText: '确认',
    cancelButtonText: '取消'
  }).catch(() => ({}))
  await adminApi.handleReport(row.id, { status: newStatus, adminRemark: remark || '' })
  if (newStatus === 'RESOLVED') {
    await adminApi.productStatus(row.productId, 'OFFLINE')
  }
  ElMessage.success('处理完成')
  await load()
}

onMounted(load)
</script>

<style scoped>
.bar { display: flex; gap: 8px; margin-bottom: 12px; }
a { color: var(--primary); text-decoration: none; }
</style>
