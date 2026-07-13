<template>
  <div class="app-container">
    <h2>我的交易</h2>
    <el-tabs v-model="role" @tab-change="load">
      <el-tab-pane label="我买的" name="buyer" />
      <el-tab-pane label="我卖的" name="seller" />
    </el-tabs>
    <el-table :data="list" v-loading="loading" border>
      <el-table-column label="商品" min-width="240">
        <template #default="{ row }">
          <div class="cell-product">
            <img v-if="row.productCover" :src="row.productCover" class="cell-img" />
            <div>
              <router-link :to="`/product/${row.productId}`">{{ row.productTitle }}</router-link>
              <div class="cell-meta">¥{{ row.productPrice }}</div>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="对方" width="120">
        <template #default="{ row }">
          {{ role === 'buyer' ? row.sellerName : row.buyerName }}
        </template>
      </el-table-column>
      <el-table-column label="见面地点" prop="meetLocation" width="160" />
      <el-table-column label="备注" prop="remark" min-width="160" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="170" prop="createdAt" />
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button v-if="role === 'seller' && row.status === 'PENDING'" size="small" type="primary" @click="op(row, 'confirm')">确认预约</el-button>
          <el-button v-if="['PENDING', 'RESERVED'].includes(row.status)" size="small" @click="op(row, 'finish')">交易完成</el-button>
          <el-button v-if="['PENDING', 'RESERVED'].includes(row.status)" size="small" type="danger" @click="op(row, 'cancel')">取消</el-button>
          <el-button size="small" @click="$router.push(`/chat/${role === 'buyer' ? row.sellerId : row.buyerId}?productId=${row.productId}`)">联系</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { orderApi } from '../api'

const role = ref('buyer')
const list = ref([])
const loading = ref(false)

const load = async () => {
  loading.value = true
  try {
    const { data } = await orderApi.list(role.value)
    list.value = data
  } finally { loading.value = false }
}

const statusText = (s) => ({ PENDING: '待确认', RESERVED: '已预约', COMPLETED: '已完成', CANCELLED: '已取消' }[s] || s)
const statusType = (s) => ({ PENDING: 'warning', RESERVED: 'primary', COMPLETED: 'success', CANCELLED: 'info' }[s] || '')

const op = async (row, action) => {
  await ElMessageBox.confirm('确认要执行该操作吗？', '提示', { type: 'warning' }).catch(() => null).then(async (ok) => {
    if (!ok) return
    if (action === 'confirm') await orderApi.confirm(row.id)
    if (action === 'finish') await orderApi.finish(row.id)
    if (action === 'cancel') await orderApi.cancel(row.id)
    ElMessage.success('操作成功')
    await load()
  })
}

onMounted(load)
</script>

<style scoped>
.cell-product { display: flex; gap: 10px; align-items: center; }
.cell-img { width: 56px; height: 56px; object-fit: cover; border-radius: 4px; background: #f5f7fa; }
.cell-meta { color: var(--price); font-size: 13px; }
a { color: #303133; text-decoration: none; }
a:hover { color: var(--primary); }
</style>
