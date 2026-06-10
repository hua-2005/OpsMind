<template>
  <div class="login-page">
    <div class="login-card">
      <h1 class="login-title">OpsMind 运维数字员工</h1>
      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="username">用户名</label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            placeholder="请输入用户名"
            required
          />
        </div>
        <div class="form-group">
          <label for="password">密码</label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            required
          />
        </div>
        <div v-if="error" class="error-message">{{ error }}</div>
        <button type="submit" class="btn-login" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { login } from '../../api/auth'

const router = useRouter()
const authStore = useAuthStore()

const form = ref({
  username: '',
  password: '',
})

const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  loading.value = true
  error.value = ''

  try {
    const res = await login(form.value)
    // Axios 拦截器已提取 response.data，所以 res = {code, message, data}
    // res.data 才是 LoginResponse（含 access_token、user、roles 等）
    const data = res.data

    authStore.setToken(data.access_token)
    authStore.setUserInfo({
      user: data.user,
      roles: data.roles,
      permissions: data.permissions,
      menus: data.menus,
    })

    // 首次登录跳转修改密码页
    if (data.user.first_login) {
      router.push('/change-password')
    } else if (data.permissions && data.permissions.length > 0) {
      // 有后台权限的用户（管理员/运维/知识库管理员）跳转后台
      router.push('/admin')
    } else {
      // 报障人无后台权限，跳转门户端
      router.push('/portal')
    }
  } catch (err: any) {
    error.value = err?.message || '登录失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--bg-base);
}

.login-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 32px;
  width: 100%;
  max-width: 400px;
}

.login-title {
  text-align: center;
  margin-bottom: 24px;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.form-group {
  margin-bottom: 16px;
}

label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  color: var(--text-secondary);
}

input {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 14px;
}

input:focus {
  outline: none;
  border-color: var(--accent);
}

.error-message {
  margin-bottom: 16px;
  padding: 8px 12px;
  background: #3a1a1a;
  color: #f87171;
  border-radius: 4px;
  font-size: 14px;
}

.btn-login {
  width: 100%;
  padding: 10px;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.btn-login:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
