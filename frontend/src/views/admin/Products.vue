<template>
  <div>
    <div class="bar">
      <el-input v-model="keyword" placeholder="搜索商品标题" clearable style="width: 240px" @keydown.enter="load" />
      <el-select v-model="status" placeholder="状态" clearable style="width: 140px">
        <el-option label="在售" value="ON_SALE" />
        <el-option label="已预约" value="RESERVED" />
        <el-option label="已售出" value="SOLD" />
        <el-option label="已下架" value="OFFLINE" />
      </el-select>
      <el-button type="primary" @click="load">查询</el-button>
    </div>
    <el-table :data="list" v-loading="loading" border>
      <el-table-column label="ID" prop="id" width="80" />
      <el-table-column label="商品">
        <template #default="{ row }">
          <div class="cell-product">
            <img v-if="row.cover" :src="row.cover" class="cell-img" />
            <router-link :to="`/product/${row.id}`">{{ row.title }}</router-link>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="价格" width="100">
        <template #default="{ row }">¥{{ row.price }}</template>
      </el-table-column>
      <el-table-column label="卖家" prop="sellerName" width="120" />
      <el-table-column label="分类" prop="categoryName" width="120" />
      <el-table-column label="状态" width="120">
        <template #default="{ row }">{{ statusText(row.status) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button size="small" type="danger" @click="setStatus(row, 'OFFLINE')" v-if="row.status !== 'OFFLINE'">下架</el-button>
          <el-button size="small" type="primary" @click="setStatus(row, 'ON_SALE')" v-else>恢复在售</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="pagination">
      <el-pagination
        background layout="prev, pager, next"
        :total="total" :page-size="size" :current-page="page"
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
const status = ref('')
const list = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)

const statusText = (s) => ({ ON_SALE: '在售', RESERVED: '已预约', SOLD: '已售出', OFFLINE: '已下架' }[s] || s)

const load = async () => {
  loading.value = true
  try {
    const { data } = await adminApi.products({
      keyword: keyword.value, status: status.value || undefined, page: page.value, size: size.value
    })
    list.value = data.records
    total.value = data.total
  } finally { loading.value = false }
}

const setStatus = async (row, s) => {
  await adminApi.productStatus(row.id, s)
  ElMessage.success('已更新')
  await load()
}

onMounted(load)
</script>

<style scoped>
.bar { display: flex; gap: 8px; margin-bottom: 12px; }
.cell-product { display: flex; gap: 10px; align-items: center; }
.cell-img { width: 40px; height: 40px; object-fit: cover; border-radius: 4px; background: #f5f7fa; }
a { color: #303133; text-decoration: none; }
a:hover { color: var(--primary); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
