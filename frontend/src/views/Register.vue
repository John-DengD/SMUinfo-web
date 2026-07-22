<template>
  <div class="auth-page">
    <div class="auth-bg">
      <span class="orb o1"></span>
      <span class="orb o2"></span>
    </div>
    <div class="auth-card">
      <div class="brand">
        <span class="brand-mark">SMU</span>
        <span class="brand-name">信息交易平台</span>
      </div>
      <h2>创建账号</h2>
      <p class="subtitle">用学号注册，加入校园闲置社区</p>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top" @submit.prevent>
        <div class="row">
          <el-form-item label="姓名" prop="name" style="flex: 1">
            <el-input v-model="form.name" placeholder="真实姓名" maxlength="20" />
          </el-form-item>
          <el-form-item label="学号" prop="studentNo" style="flex: 1">
            <el-input v-model="form.studentNo" placeholder="12 位数字学号" maxlength="12" inputmode="numeric" />
          </el-form-item>
        </div>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
        <div class="row">
          <el-form-item label="手机" style="flex: 1">
            <el-input v-model="form.phone" placeholder="可选" />
          </el-form-item>
          <el-form-item label="学院" style="flex: 1">
            <el-input v-model="form.college" placeholder="例如：计算机学院" />
          </el-form-item>
        </div>
        <el-form-item label="校区">
          <el-input v-model="form.campus" placeholder="例如：主校区" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" :loading="loading" @click="submit" style="width: 100%">注册</el-button>
        </el-form-item>
      </el-form>
      <div class="alt">
        已有账号？<router-link to="/login">直接登录</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { identify, track, cv } from '@hellyeah/x-ray'
import { authApi } from '../api'

const router = useRouter()
const formRef = ref(null)
const loading = ref(false)
const form = reactive({ name: '', studentNo: '', password: '', phone: '', college: '', campus: '' })
const fakeStudentNos = new Set(['000000000000', '111111111111', '123456789012'])
const namePattern = /^[\u4e00-\u9fa5A-Za-z·.\- ]+$/
const studentNoPattern = /^\d{12}$/

const normalizeForm = () => {
  form.name = form.name.trim().replace(/\s+/g, ' ')
  form.studentNo = form.studentNo.trim()
}

const validateName = (_, value, callback) => {
  const name = (value || '').trim().replace(/\s+/g, ' ')
  if (!name) return callback(new Error('请输入姓名'))
  if (name.length < 2 || name.length > 20) return callback(new Error('姓名长度需为 2-20 个字符'))
  if (!namePattern.test(name)) return callback(new Error('姓名不能包含数字或特殊字符'))
  return callback()
}

const validateStudentNo = (_, value, callback) => {
  const studentNo = (value || '').trim()
  if (!studentNo) return callback(new Error('请输入学号'))
  if (!studentNoPattern.test(studentNo)) return callback(new Error('学号必须是 12 位纯数字'))
  if (fakeStudentNos.has(studentNo)) return callback(new Error('请填写真实学号'))
  return callback()
}

const rules = {
  name: [{ validator: validateName, trigger: 'blur' }],
  studentNo: [{ validator: validateStudentNo, trigger: 'blur' }],
  password: [{ required: true, min: 6, message: '密码至少 6 位', trigger: 'blur' }]
}
const submit = async () => {
  if (!formRef.value) return
  normalizeForm()
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      const { data } = await authApi.register(form)
      identify(String(data.id))
      track(cv.registrationComplete, { signup_method: 'student_no' })
      ElMessage.success('注册成功，请登录')
      router.push('/login')
    } catch (e) { /* ignore */ } finally { loading.value = false }
  })
}
</script>

<style scoped>
.auth-page {
  position: relative;
  min-height: calc(100vh - 130px);
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 40px 16px;
  overflow: hidden;
}
.auth-bg { position: absolute; inset: 0; pointer-events: none; z-index: 0; }
.orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(60px);
  opacity: 0.55;
}
.o1 { width: 320px; height: 320px; background: #c4b5fd; top: -80px; right: -60px; }
.o2 { width: 280px; height: 280px; background: #67e8f9; bottom: -60px; left: -40px; }
.auth-card {
  position: relative;
  z-index: 1;
  background: rgba(255,255,255,0.92);
  backdrop-filter: blur(20px);
  padding: 32px 36px 24px;
  border-radius: 18px;
  width: 520px;
  max-width: 100%;
  box-shadow: 0 24px 48px rgba(79, 70, 229, 0.18);
  border: 1px solid rgba(255,255,255,0.6);
}
.brand { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; }
.brand-mark {
  width: 46px; height: 36px; border-radius: 10px;
  background: linear-gradient(135deg, var(--primary), var(--accent));
  color: #fff; display: inline-flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 13px;
}
.brand-name { font-weight: 600; color: #111827; }
.auth-card h2 { margin: 0 0 6px 0; font-size: 22px; color: #111827; }
.subtitle { margin: 0 0 18px 0; color: var(--text-muted); font-size: 13px; }
.row { display: flex; gap: 12px; }
.alt { text-align: center; color: var(--text-muted); font-size: 13px; }
@media (max-width: 560px) {
  .row { flex-direction: column; gap: 0; }
}
</style>
