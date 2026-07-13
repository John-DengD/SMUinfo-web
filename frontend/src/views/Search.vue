<template>
  <div class="app-container">
    <div class="filter-bar">
      <el-form :model="form" inline @submit.prevent>
        <el-form-item label="关键词">
          <el-input v-model="form.keyword" placeholder="商品名称" clearable />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.categoryId" placeholder="全部" clearable style="width: 140px">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="价格">
          <el-input v-model.number="form.minPrice" placeholder="最低" style="width: 90px" />
          <span style="margin: 0 4px">-</span>
          <el-input v-model.number="form.maxPrice" placeholder="最高" style="width: 90px" />
        </el-form-item>
        <el-form-item label="成色">
          <el-select v-model="form.conditionLevel" placeholder="全部" clearable style="width: 120px">
            <el-option label="全新" value="全新" />
            <el-option label="九成新" value="九成新" />
            <el-option label="八成新" value="八成新" />
            <el-option label="七成新" value="七成新" />
            <el-option label="其他" value="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="校区">
          <el-input v-model="form.campus" placeholder="校区或地点" clearable style="width: 140px" />
        </el-form-item>
        <el-form-item label="排序">
          <el-select v-model="form.sortBy" style="width: 120px">
            <el-option label="最新" value="" />
            <el-option label="价格升序" value="price_asc" />
            <el-option label="价格降序" value="price_desc" />
            <el-option label="热度" value="hot" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">搜索</el-button>
          <el-button @click="reset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="product-grid" v-loading="loading">
      <ProductCard v-for="p in products" :key="p.id" :item="p" />
    </div>
    <div v-if="!loading && products.length === 0" class="empty-block">没有找到匹配的商品</div>

    <div class="pagination" v-if="total > size">
      <el-pagination
        background
        layout="prev, pager, next"
        :total="total"
        :page-size="size"
        :current-page="page"
        @current-change="onPage"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { categoryApi, productApi } from '../api'
import ProductCard from '../components/ProductCard.vue'

const route = useRoute()
const form = reactive({
  keyword: route.query.keyword || '',
  categoryId: null,
  minPrice: null,
  maxPrice: null,
  conditionLevel: '',
  campus: '',
  sortBy: ''
})
const categories = ref([])
const products = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(12)
const loading = ref(false)

const search = async () => {
  page.value = 1
  await load()
}
const reset = () => {
  form.keyword = ''
  form.categoryId = null
  form.minPrice = null
  form.maxPrice = null
  form.conditionLevel = ''
  form.campus = ''
  form.sortBy = ''
  search()
}
const load = async () => {
  loading.value = true
  try {
    const { data } = await productApi.list({
      page: page.value,
      size: size.value,
      ...form
    })
    products.value = data.records
    total.value = data.total
  } finally {
    loading.value = false
  }
}
const onPage = (p) => { page.value = p; load() }

onMounted(async () => {
  const c = await categoryApi.list()
  categories.value = c.data
  await load()
})

watch(() => route.query.keyword, (v) => {
  form.keyword = v || ''
  search()
})
</script>

<style scoped>
.filter-bar {
  background: #fff;
  padding: 16px 16px 0 16px;
  border-radius: 8px;
  margin-bottom: 16px;
}
.pagination { display: flex; justify-content: center; margin: 24px 0; }
</style>
