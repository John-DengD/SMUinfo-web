<template>
  <div class="product-card" @click="$router.push(`/product/${item.id}`)">
    <div class="cover">
      <img v-if="item.cover" :src="item.cover" />
      <span v-else style="color:#c0c4cc">暂无图片</span>
      <span v-if="item.status !== 'ON_SALE'" class="status-overlay">{{ statusText }}</span>
    </div>
    <div class="body">
      <div class="title">{{ item.title }}</div>
      <div class="price">¥{{ item.price }}</div>
      <div class="meta">
        <span>{{ item.conditionLevel || '九成新' }}</span>
        <span>{{ item.tradeLocation || item.sellerCampus || '校园' }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ item: { type: Object, required: true } })
const statusMap = { RESERVED: '已预约', SOLD: '已售出', OFFLINE: '已下架' }
const statusText = computed(() => statusMap[props.item.status] || '')
</script>

<style scoped>
.cover { position: relative; }
.status-overlay {
  position: absolute;
  top: 8px;
  left: 8px;
  background: rgba(0,0,0,0.6);
  color: #fff;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
}
</style>
