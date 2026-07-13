<template>
  <div class="app-container" v-loading="loading">
    <div v-if="product" class="detail">
      <div class="gallery">
        <el-carousel v-if="product.images && product.images.length" height="420px">
          <el-carousel-item v-for="(img, idx) in product.images" :key="idx">
            <img :src="img" class="g-img" />
          </el-carousel-item>
        </el-carousel>
        <div v-else class="no-image">暂无图片</div>
      </div>
      <div class="info">
        <h2>{{ product.title }}</h2>
        <div class="price-row">
          <span class="price">¥{{ product.price }}</span>
          <span v-if="product.originalPrice" class="origin">原价 ¥{{ product.originalPrice }}</span>
          <span class="status status-tag" :class="statusClass">{{ statusText }}</span>
        </div>
        <div class="meta-row">
          <span>成色：{{ product.conditionLevel || '-' }}</span>
          <span>交易地点：{{ product.tradeLocation || '-' }}</span>
          <span>分类：{{ product.categoryName }}</span>
          <span>浏览：{{ product.viewCount }}</span>
        </div>
        <div class="desc">{{ product.description || '卖家很懒，没有写描述' }}</div>
        <div class="seller">
          <div class="seller-info">
            <div class="avatar">{{ (product.sellerName || '?').slice(0, 1) }}</div>
            <div>
              <div>{{ product.sellerName }}</div>
              <div class="campus">{{ product.sellerCampus || '校园' }}</div>
            </div>
          </div>
        </div>

        <div class="actions">
          <el-button :type="product.favorited ? 'default' : 'primary'" plain @click="toggleFavorite">
            {{ product.favorited ? '已收藏' : '收藏' }}
          </el-button>
          <el-button type="primary" @click="contactSeller">联系卖家</el-button>
          <el-button type="warning" :disabled="!canWant" @click="wantDialog = true">我想要</el-button>
          <el-button @click="reportDialog = true">举报</el-button>
        </div>
      </div>
    </div>

    <el-dialog v-model="wantDialog" title="发起线下交易请求" width="420px">
      <el-form :model="wantForm" label-width="80px">
        <el-form-item label="见面地点">
          <el-input v-model="wantForm.meetLocation" placeholder="例如：主校区图书馆门口" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="wantForm.remark" type="textarea" :rows="3" placeholder="例如：周六下午方便" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="wantDialog = false">取消</el-button>
        <el-button type="primary" @click="submitWant">提交</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="reportDialog" title="举报商品" width="420px">
      <el-input v-model="reportReason" type="textarea" :rows="4" placeholder="请描述违规情况" />
      <template #footer>
        <el-button @click="reportDialog = false">取消</el-button>
        <el-button type="primary" @click="submitReport">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { productApi, favoriteApi, orderApi, reportApi } from '../api'
import { useUserStore } from '../stores/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const product = ref(null)
const loading = ref(false)
const wantDialog = ref(false)
const reportDialog = ref(false)
const wantForm = ref({ meetLocation: '', remark: '' })
const reportReason = ref('')

const statusMap = { ON_SALE: '在售', RESERVED: '已预约', SOLD: '已售出', OFFLINE: '已下架' }
const statusText = computed(() => statusMap[product.value?.status] || '')
const statusClass = computed(() => {
  const m = { ON_SALE: 'on-sale', RESERVED: 'reserved', SOLD: 'sold', OFFLINE: 'offline' }
  return m[product.value?.status]
})
const canWant = computed(() => {
  if (!product.value) return false
  if (product.value.status !== 'ON_SALE') return false
  if (!userStore.isLoggedIn) return true
  return product.value.sellerId !== userStore.user.id
})

const load = async () => {
  loading.value = true
  try {
    const { data } = await productApi.detail(route.params.id)
    product.value = data
  } finally {
    loading.value = false
  }
}

const requireLogin = () => {
  if (!userStore.isLoggedIn) {
    router.push({ path: '/login', query: { redirect: route.fullPath } })
    return false
  }
  return true
}

const toggleFavorite = async () => {
  if (!requireLogin()) return
  if (product.value.favorited) {
    await favoriteApi.remove(product.value.id)
    product.value.favorited = false
    ElMessage.success('已取消收藏')
  } else {
    await favoriteApi.add(product.value.id)
    product.value.favorited = true
    ElMessage.success('收藏成功')
  }
}

const contactSeller = () => {
  if (!requireLogin()) return
  if (product.value.sellerId === userStore.user.id) {
    ElMessage.info('这是你自己发布的商品')
    return
  }
  router.push({ path: `/chat/${product.value.sellerId}`, query: { productId: product.value.id } })
}

const submitWant = async () => {
  if (!requireLogin()) return
  try {
    await orderApi.create({
      productId: product.value.id,
      meetLocation: wantForm.value.meetLocation,
      remark: wantForm.value.remark
    })
    wantDialog.value = false
    ElMessage.success('交易请求已发起，可在「我的交易」查看进度')
    await load()
  } catch (e) { /* api 已提示 */ }
}

const submitReport = async () => {
  if (!requireLogin()) return
  if (!reportReason.value.trim()) {
    ElMessage.warning('请填写举报原因')
    return
  }
  try {
    await reportApi.create({ productId: product.value.id, reason: reportReason.value })
    reportDialog.value = false
    reportReason.value = ''
    ElMessage.success('举报已提交，等待管理员处理')
  } catch (e) { /* ignore */ }
}

onMounted(load)
</script>

<style scoped>
.detail {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  display: grid;
  grid-template-columns: 480px 1fr;
  gap: 24px;
}
.no-image {
  background: #f5f7fa;
  width: 100%;
  height: 420px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  border-radius: 4px;
}
.g-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.info h2 { margin: 0 0 12px; }
.price-row {
  display: flex;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 12px;
}
.price { font-size: 26px; color: var(--price); font-weight: 600; }
.origin { color: #909399; text-decoration: line-through; font-size: 13px; }
.status { font-size: 13px; }
.meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  color: #606266;
  font-size: 13px;
  margin-bottom: 16px;
}
.desc {
  background: #fafafa;
  padding: 12px;
  border-radius: 6px;
  line-height: 1.6;
  margin-bottom: 16px;
  white-space: pre-wrap;
}
.seller {
  border-top: 1px dashed #ebeef5;
  padding-top: 14px;
  margin-bottom: 16px;
}
.seller-info { display: flex; gap: 10px; align-items: center; }
.avatar {
  width: 36px; height: 36px; border-radius: 50%;
  background: var(--primary); color: #fff;
  display: flex; align-items: center; justify-content: center;
}
.campus { color: #909399; font-size: 12px; }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
@media (max-width: 720px) {
  .detail { grid-template-columns: 1fr; }
}
</style>
