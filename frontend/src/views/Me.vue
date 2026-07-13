<template>
  <div class="app-container">
    <div class="card" v-if="form">
      <h2>我的资料</h2>
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" maxlength="20" />
        </el-form-item>
        <el-form-item label="学号">
          <el-input v-model="form.studentNo" disabled />
        </el-form-item>
        <el-form-item label="手机">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item label="学院">
          <el-input v-model="form.college" />
        </el-form-item>
        <el-form-item label="校区">
          <el-input v-model="form.campus" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="save">保存</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { authApi } from '../api'
import { useUserStore } from '../stores/user'

const form = ref(null)
const formRef = ref(null)
const userStore = useUserStore()
const namePattern = /^[\u4e00-\u9fa5A-Za-z·.\- ]+$/

const validateName = (_, value, callback) => {
  const name = (value || '').trim().replace(/\s+/g, ' ')
  if (!name) return callback(new Error('请输入姓名'))
  if (name.length < 2 || name.length > 20) return callback(new Error('姓名长度需为 2-20 个字符'))
  if (!namePattern.test(name)) return callback(new Error('姓名不能包含数字或特殊字符'))
  return callback()
}

const rules = {
  name: [{ validator: validateName, trigger: 'blur' }]
}

onMounted(async () => {
  const { data } = await authApi.me()
  form.value = data
})
const save = async () => {
  if (!formRef.value) return
  form.value.name = form.value.name.trim().replace(/\s+/g, ' ')
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    const { data } = await authApi.updateMe(form.value)
    ElMessage.success('保存成功')
    userStore.user = data
    localStorage.setItem('user', JSON.stringify(data))
  })
}
</script>

<style scoped>
.card { background: #fff; padding: 24px; border-radius: 8px; max-width: 640px; margin: 0 auto; }
.card h2 { margin-top: 0; }
</style>
