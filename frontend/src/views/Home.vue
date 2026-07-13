<template>
  <div class="app-container home">
    <section class="banner">
      <div class="banner-text">
        <div class="kicker">SMU 校园社区</div>
        <h2>校内闲置，安心流转</h2>
        <p>发布闲置 · 浏览好物 · 同校园区线下面交</p>
        <div class="banner-actions">
          <router-link to="/publish" class="btn-primary">我要发布</router-link>
          <router-link to="/feedback" class="btn-ghost">提点建议</router-link>
        </div>
      </div>
      <div class="banner-deco">
        <div class="bubble b1">📚</div>
        <div class="bubble b2">💻</div>
        <div class="bubble b3">🎒</div>
        <div class="bubble b4">🚲</div>
      </div>
    </section>

    <section class="categories surface" :class="{ expanded: categoryExpanded }">
      <div class="category-head">
        <h3 class="section-title">商品分类</h3>
        <button class="category-toggle" type="button" @click="toggleCategoryExpanded">
          <span>{{ currentCategoryName }}</span>
          <el-icon :class="{ open: categoryExpanded }"><ArrowDown /></el-icon>
        </button>
      </div>
      <div class="cat-grid" :class="{ open: categoryExpanded }">
        <div
          class="cat-item"
          :class="{ active: !currentCategory }"
          @click="selectCategory(null)"
        >
          <div class="cat-icon all">全</div>
          <div class="cat-name">全部</div>
        </div>
        <div
          v-for="c in categories"
          :key="c.id"
          class="cat-item"
          :class="{ active: currentCategory === c.id }"
          @click="selectCategory(c.id)"
        >
          <div class="cat-icon">{{ (c.name || '?').slice(0,1) }}</div>
          <div class="cat-name">{{ c.name }}</div>
        </div>
      </div>
    </section>

    <section class="sort">
      <span class="sort-label">排序</span>
      <el-radio-group v-model="sortBy" size="small" @change="loadProducts(true)">
        <el-radio-button label="">最新</el-radio-button>
        <el-radio-button label="price_asc">价格 ↑</el-radio-button>
        <el-radio-button label="price_desc">价格 ↓</el-radio-button>
        <el-radio-button label="hot">热度</el-radio-button>
      </el-radio-group>
    </section>

    <section class="product-grid" v-loading="loading">
      <ProductCard v-for="p in products" :key="p.id" :item="p" />
    </section>
    <div v-if="!loading && products.length === 0" class="empty-block">
      <div style="font-size: 40px; margin-bottom: 12px;">🛍️</div>
      <div>暂无商品，<router-link to="/publish" style="color: var(--primary)">去发布第一件吧</router-link></div>
    </div>

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
import { computed, ref, onMounted } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'
import { categoryApi, productApi } from '../api'
import ProductCard from '../components/ProductCard.vue'

const categories = ref([])
const products = ref([])
const total = ref(0)
const page = ref(1)
const size = ref(12)
const loading = ref(false)
const currentCategory = ref(null)
const sortBy = ref('')
const categoryExpanded = ref(false)

const currentCategoryName = computed(() => {
  if (!currentCategory.value) return '全部'
  return categories.value.find(c => c.id === currentCategory.value)?.name || '全部'
})

const loadCategories = async () => {
  const { data } = await categoryApi.list()
  categories.value = data
}

const loadProducts = async (reset = false) => {
  if (reset) page.value = 1
  loading.value = true
  try {
    const { data } = await productApi.list({
      page: page.value,
      size: size.value,
      categoryId: currentCategory.value,
      sortBy: sortBy.value
    })
    products.value = data.records
    total.value = data.total
  } finally {
    loading.value = false
  }
}

const selectCategory = (id) => {
  currentCategory.value = id
  categoryExpanded.value = false
  loadProducts(true)
}

const toggleCategoryExpanded = () => {
  categoryExpanded.value = !categoryExpanded.value
}

const onPage = (p) => {
  page.value = p
  loadProducts()
}

onMounted(async () => {
  await loadCategories()
  await loadProducts()
})
</script>

<style scoped>
.home { padding-top: 20px; }
.banner {
  position: relative;
  background: linear-gradient(135deg, #4f46e5 0%, #6366f1 40%, #06b6d4 100%);
  color: #fff;
  border-radius: 20px;
  padding: 40px 32px;
  margin-bottom: 20px;
  overflow: hidden;
  box-shadow: 0 18px 40px rgba(79, 70, 229, 0.25);
}
.banner .kicker {
  display: inline-block;
  padding: 4px 10px;
  background: rgba(255,255,255,0.18);
  border-radius: 999px;
  font-size: 12px;
  letter-spacing: 1px;
  margin-bottom: 12px;
}
.banner h2 { margin: 0 0 8px 0; font-size: 30px; letter-spacing: 0.5px; font-weight: 700; }
.banner p { margin: 0 0 18px 0; opacity: 0.92; font-size: 14px; }
.banner-actions { display: flex; gap: 12px; }
.btn-primary, .btn-ghost {
  display: inline-block;
  padding: 9px 18px;
  border-radius: 999px;
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  transition: transform .15s, background .15s;
}
.btn-primary {
  background: #fff;
  color: var(--primary);
}
.btn-ghost {
  background: rgba(255,255,255,0.15);
  color: #fff;
  border: 1px solid rgba(255,255,255,0.35);
}
.btn-primary:hover, .btn-ghost:hover { transform: translateY(-1px); }

.banner-deco {
  position: absolute;
  right: 24px;
  top: 0;
  bottom: 0;
  width: 280px;
  pointer-events: none;
}
.bubble {
  position: absolute;
  background: rgba(255,255,255,0.18);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  backdrop-filter: blur(4px);
}
.b1 { width: 70px; height: 70px; top: 24px; right: 40px; }
.b2 { width: 56px; height: 56px; top: 90px; right: 130px; }
.b3 { width: 64px; height: 64px; bottom: 30px; right: 70px; }
.b4 { width: 48px; height: 48px; bottom: 60px; right: 180px; }

.categories {
  padding: 18px 18px 14px;
  margin-bottom: 18px;
}
.category-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.category-toggle {
  display: none;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 112px;
  max-width: 48%;
  height: 34px;
  padding: 0 10px 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
  color: #374151;
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}
.category-toggle span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.category-toggle .el-icon {
  flex: 0 0 auto;
  transition: transform .18s ease;
}
.category-toggle .el-icon.open { transform: rotate(180deg); }
.cat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(86px, 1fr));
  gap: 8px;
}
.cat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 6px;
  border-radius: 10px;
  cursor: pointer;
  transition: background .15s, transform .15s;
  font-size: 13px;
  color: #4b5563;
}
.cat-item:hover { background: var(--primary-soft); transform: translateY(-1px); }
.cat-item.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.cat-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, #eef2ff, #e0f2fe);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  color: var(--primary);
  font-size: 16px;
}
.cat-item.active .cat-icon {
  background: linear-gradient(135deg, var(--primary), var(--accent));
  color: #fff;
}
.cat-icon.all { background: linear-gradient(135deg, var(--primary), var(--accent)); color: #fff; }
.sort {
  background: #fff;
  border-radius: var(--radius);
  padding: 12px 16px;
  margin-bottom: 18px;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 10px;
  box-shadow: var(--shadow-sm);
}
.sort .sort-label { color: var(--text-muted); }
.pagination { display: flex; justify-content: center; margin: 28px 0 16px; }

@media (max-width: 720px) {
  .banner { padding: 28px 20px; border-radius: 16px; }
  .banner h2 { font-size: 22px; }
  .banner-deco { display: none; }
  .categories {
    padding: 12px;
    margin-bottom: 12px;
  }
  .categories .section-title { margin: 0; }
  .category-toggle { display: inline-flex; }
  .cat-grid {
    display: flex;
    gap: 8px;
    margin-top: 0;
    max-height: 0;
    overflow: hidden;
    opacity: 0;
    pointer-events: none;
    transition: max-height .2s ease, margin-top .2s ease, opacity .16s ease;
  }
  .cat-grid.open {
    margin-top: 12px;
    max-height: 168px;
    overflow-x: auto;
    overflow-y: hidden;
    opacity: 1;
    pointer-events: auto;
    padding-bottom: 4px;
  }
  .cat-item {
    flex: 0 0 72px;
    gap: 6px;
    padding: 10px 6px;
    border-radius: 8px;
  }
  .cat-icon {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    font-size: 14px;
  }
  .cat-name {
    width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: center;
  }
  .sort {
    align-items: flex-start;
    gap: 8px;
    padding: 10px 12px;
    margin-bottom: 12px;
    overflow-x: auto;
  }
  .sort .sort-label {
    flex: 0 0 auto;
    line-height: 28px;
  }
}
</style>
