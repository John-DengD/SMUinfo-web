<template>
  <div class="app-container">
    <h2>我的收藏</h2>
    <div class="product-grid" v-loading="loading">
      <ProductCard v-for="p in list" :key="p.id" :item="p" />
    </div>
    <div v-if="!loading && list.length === 0" class="empty-block">还没有收藏的商品</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { favoriteApi } from '../api'
import ProductCard from '../components/ProductCard.vue'

const list = ref([])
const loading = ref(false)
onMounted(async () => {
  loading.value = true
  try {
    const { data } = await favoriteApi.list()
    list.value = data.records
  } finally { loading.value = false }
})
</script>
