<script setup lang="ts">
// 登录页面
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!username.value || !password.value) {
    ElMessage.warning('请输入账号和密码')
    return
  }
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    await auth.loadPermissions()
    router.push('/admin')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string } } } }
    const msg = err?.response?.data?.error?.message || '登录失败'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <form class="login-card" @submit.prevent="handleLogin">
      <h1 class="login-title">OpsMind</h1>
      <p class="login-desc">运维数字员工系统</p>
      <div class="login-field">
        <label>账号</label>
        <input v-model="username" type="text" placeholder="请输入账号" autocomplete="username" />
      </div>
      <div class="login-field">
        <label>密码</label>
        <input v-model="password" type="password" placeholder="请输入密码" autocomplete="current-password" />
      </div>
      <button type="submit" class="login-btn" :disabled="loading">
        {{ loading ? '登录中...' : '登 录' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #08090a;
}
.login-card {
  width: 360px;
  padding: 40px 32px;
  background: #0f1011;
  border: 1px solid rgba(255,255,255,0.05);
  border-radius: 12px;
}
.login-title {
  font-size: 28px;
  font-weight: 590;
  letter-spacing: -0.8px;
  color: #f7f8f8;
  margin: 0 0 4px;
}
.login-desc {
  color: #8a8f98;
  font-size: 14px;
  margin: 0 0 28px;
}
.login-field {
  margin-bottom: 16px;
}
.login-field label {
  display: block;
  color: #8a8f98;
  font-size: 13px;
  margin-bottom: 6px;
}
.login-field input {
  width: 100%;
  padding: 10px 12px;
  background: #191a1b;
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 8px;
  color: #f7f8f8;
  font-size: 14px;
  outline: none;
  box-sizing: border-box;
}
.login-field input:focus {
  border-color: #5e6ad2;
}
.login-btn {
  width: 100%;
  padding: 12px;
  margin-top: 8px;
  background: #5e6ad2;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 510;
  cursor: pointer;
  transition: background 0.15s;
}
.login-btn:hover { background: #7170ff; }
.login-btn:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
