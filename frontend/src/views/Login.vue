<template>
  <div class="auth-page">
    <div class="auth-bg">
      <span class="orb o1"></span>
      <span class="orb o2"></span>
      <span class="orb o3"></span>
    </div>
    <div class="auth-card">
      <div class="brand">
        <span class="brand-mark">SMU</span>
        <span class="brand-name">信息交易平台</span>
      </div>
      <h2>欢迎回来</h2>
      <p class="subtitle">登录后浏览闲置、发布商品、与同学聊起来</p>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top" @submit.prevent>
        <el-form-item label="学号" prop="studentNo">
          <el-input v-model="form.studentNo" size="large" placeholder="请输入学号" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" size="large" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" :loading="loading" @click="submit" style="width: 100%">登录</el-button>
        </el-form-item>
      </el-form>
      <div class="alt">
        没有账号？<router-link to="/register">立即注册</router-link>
      </div>
      <div class="hint">演示账号：admin / admin123 ・ student001 / 123456</div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '../stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const formRef = ref(null)
const loading = ref(false)
const form = reactive({ studentNo: '', password: '' })
const rules = {
  studentNo: [{ required: true, message: '请输入学号' }],
  password: [{ required: true, message: '请输入密码' }]
}
const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      await userStore.login(form)
      ElMessage.success('登录成功')
      router.replace(route.query.redirect || '/')
    } catch (e) { /* api 已提示 */ } finally { loading.value = false }
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
.o1 { width: 320px; height: 320px; background: #a5b4fc; top: -80px; left: -60px; }
.o2 { width: 280px; height: 280px; background: #67e8f9; bottom: -60px; right: -40px; }
.o3 { width: 200px; height: 200px; background: #c4b5fd; top: 40%; right: 30%; opacity: 0.4; }
.auth-card {
  position: relative;
  z-index: 1;
  background: rgba(255,255,255,0.92);
  backdrop-filter: blur(20px);
  padding: 36px 36px 28px;
  border-radius: 18px;
  width: 440px;
  max-width: 100%;
  box-shadow: 0 24px 48px rgba(79, 70, 229, 0.18);
  border: 1px solid rgba(255,255,255,0.6);
}
.brand { display: flex; align-items: center; gap: 10px; margin-bottom: 18px; }
.brand-mark {
  width: 46px; height: 36px; border-radius: 10px;
  background: linear-gradient(135deg, var(--primary), var(--accent));
  color: #fff; display: inline-flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 13px;
}
.brand-name { font-weight: 600; color: #111827; }
.auth-card h2 { margin: 0 0 6px 0; font-size: 22px; color: #111827; }
.subtitle { margin: 0 0 22px 0; color: var(--text-muted); font-size: 13px; }
.alt { text-align: center; color: var(--text-muted); font-size: 13px; margin-top: 4px; }
.hint {
  margin-top: 14px;
  text-align: center;
  font-size: 12px;
  color: #9ca3af;
  background: #f9fafb;
  padding: 8px;
  border-radius: 8px;
}
</style>
