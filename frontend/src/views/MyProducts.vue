<template>
  <div class="app-container">
    <div class="head">
      <h2>我的发布</h2>
      <el-button type="primary" @click="$router.push('/publish')">+ 发布闲置</el-button>
    </div>
    <el-tabs v-model="tab" @tab-change="load">
      <el-tab-pane label="全部" name="" />
      <el-tab-pane label="在售" name="ON_SALE" />
      <el-tab-pane label="已预约" name="RESERVED" />
      <el-tab-pane label="已售出" name="SOLD" />
      <el-tab-pane label="已下架" name="OFFLINE" />
    </el-tabs>

    <el-table :data="list" v-loading="loading" border>
      <el-table-column label="商品">
        <template #default="{ row }">
          <div class="cell-product">
            <img v-if="row.cover" :src="row.cover" class="cell-img" />
            <div class="cell-info">
              <router-link :to="`/product/${row.id}`">{{ row.title }}</router-link>
              <div class="cell-meta">¥{{ row.price }} · {{ row.conditionLevel || '-' }}</div>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="浏览" prop="viewCount" width="80" />
      <el-table-column label="发布时间" width="180">
        <template #default="{ row }">{{ row.createdAt }}</template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button size="small" @click="$router.push(`/publish/${row.id}`)">编辑</el-button>
          <el-button v-if="row.status === 'ON_SALE'" size="small" type="warning" @click="changeStatus(row, 'OFFLINE')">下架</el-button>
          <el-button v-if="row.status === 'OFFLINE'" size="small" type="primary" @click="changeStatus(row, 'ON_SALE')">上架</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { productApi } from '../api'
import { useUserStore } from '../stores/user'

const userStore = useUserStore()
const tab = ref('')
const list = ref([])
const loading = ref(false)

const load = async () => {
  loading.value = true
  try {
    const { data } = await productApi.list({
      sellerId: userStore.user.id,
      status: tab.value,
      page: 1,
      size: 100
    })
    list.value = data.records
  } finally {
    loading.value = false
  }
}

const statusText = (s) => ({ ON_SALE: '在售', RESERVED: '已预约', SOLD: '已售出', OFFLINE: '已下架' }[s] || s)
const statusType = (s) => ({ ON_SALE: 'success', RESERVED: 'warning', SOLD: 'info', OFFLINE: 'danger' }[s] || '')

const changeStatus = async (row, status) => {
  await productApi.update(row.id, { status })
  ElMessage.success('已更新')
  await load()
}

onMounted(load)
</script>

<style scoped>
.head { display: flex; justify-content: space-between; align-items: center; }
.cell-product { display: flex; gap: 10px; align-items: center; }
.cell-img { width: 56px; height: 56px; object-fit: cover; border-radius: 4px; background: #f5f7fa; }
.cell-info a { color: #303133; text-decoration: none; }
.cell-info a:hover { color: var(--primary); }
.cell-meta { color: #909399; font-size: 12px; margin-top: 4px; }
</style>
