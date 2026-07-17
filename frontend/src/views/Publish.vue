<template>
  <div class="app-container">
    <div class="card">
      <h2>{{ editId ? '编辑商品' : '发布闲置' }}</h2>
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px" @submit.prevent>
        <el-form-item label="商品标题" prop="title">
          <el-input v-model="form.title" placeholder="一句话说清楚商品" maxlength="80" show-word-limit />
        </el-form-item>
        <el-form-item label="分类" prop="categoryId">
          <el-select v-model="form.categoryId" placeholder="请选择">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="价格" prop="price">
          <el-input-number v-model="form.price" :min="0" :precision="2" />
        </el-form-item>
        <el-form-item label="原价">
          <el-input-number v-model="form.originalPrice" :min="0" :precision="2" />
        </el-form-item>
        <el-form-item label="商品成色">
          <el-select v-model="form.conditionLevel" clearable>
            <el-option label="全新" value="全新" />
            <el-option label="九成新" value="九成新" />
            <el-option label="八成新" value="八成新" />
            <el-option label="七成新" value="七成新" />
            <el-option label="其他" value="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="交易地点">
          <el-input v-model="form.tradeLocation" placeholder="例如：主校区图书馆门口" />
        </el-form-item>
        <el-form-item label="商品描述">
          <el-input v-model="form.description" type="textarea" :rows="4" placeholder="说明商品状况、瑕疵、配件等" maxlength="2000" show-word-limit />
        </el-form-item>
        <el-form-item label="商品图片">
          <el-upload
            :show-file-list="true"
            :file-list="fileList"
            :http-request="customUpload"
            :on-remove="onRemove"
            list-type="picture-card"
            accept="image/*"
            :limit="9"
          >
            <el-icon><Plus /></el-icon>
          </el-upload>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="submit">{{ editId ? '保存' : '发布' }}</el-button>
          <el-button @click="$router.back()">取消</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { categoryApi, productApi, uploadApi } from '../api'

const route = useRoute()
const router = useRouter()
const formRef = ref(null)
const loading = ref(false)
const categories = ref([])
const fileList = ref([])
const editId = ref(route.params.id || null)

const form = reactive({
  title: '',
  categoryId: null,
  price: 0,
  originalPrice: null,
  conditionLevel: '九成新',
  tradeLocation: '',
  description: '',
  images: []
})

const rules = {
  title: [{ required: true, message: '请输入商品标题' }],
  categoryId: [{ required: true, message: '请选择分类' }],
  price: [{ required: true, message: '请输入价格', type: 'number' }]
}

const customUpload = async ({ file, onSuccess, onError }) => {
  const fd = new FormData()
  fd.append('file', file)
  try {
    const { data } = await uploadApi.image(fd)
    form.images.push(data.url)
    fileList.value.push({ name: file.name, url: data.url })
    onSuccess && onSuccess(data)
  } catch (e) {
    onError && onError(e)
  }
}

const onRemove = (file) => {
  form.images = form.images.filter(u => u !== file.url)
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      if (editId.value) {
        await productApi.update(editId.value, form)
        ElMessage.success('保存成功')
      } else {
        await productApi.create(form)
        ElMessage.success('发布成功')
      }
      router.push('/my/products')
    } catch (e) { /* ignore */ } finally { loading.value = false }
  })
}

onMounted(async () => {
  const c = await categoryApi.list()
  categories.value = c.data
  if (editId.value) {
    const { data } = await productApi.detail(editId.value)
    form.title = data.title
    form.categoryId = data.categoryId
    form.price = Number(data.price)
    form.originalPrice = data.originalPrice ? Number(data.originalPrice) : null
    form.conditionLevel = data.conditionLevel
    form.tradeLocation = data.tradeLocation
    form.description = data.description
    form.images = [...(data.images || [])]
    fileList.value = (data.images || []).map(u => ({ name: u, url: u }))
  }
})
</script>

<style scoped>
.card { background: #fff; padding: 24px; border-radius: 8px; max-width: 760px; margin: 0 auto; }
.card h2 { margin-top: 0; }
</style>
