<script setup lang="ts">
// 账号创建/编辑页面
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { createAccount, getAccountDetail, updateAccount } from '@/api/accounts'
import { getRoles } from '@/api/roles'
import type { RoleItem } from '@/api/roles'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => route.params.id !== undefined && route.params.id !== 'create')
const accountId = computed(() => Number(route.params.id))

const form = ref({ username: '', password: '', real_name: '', phone: '', email: '', role_ids: [] as number[], remark: '' })
const roles = ref<RoleItem[]>([])
const loading = ref(false)

async function loadRoles() {
  const res = await getRoles({ per_page: 100 })
  roles.value = res.data.data
}

async function loadDetail() {
  if (!isEdit.value) return
  const res = await getAccountDetail(accountId.value)
  const d = res.data.data
  form.value = {
    username: '',
    password: '',
    real_name: d.real_name,
    phone: d.phone || '',
    email: d.email || '',
    role_ids: d.roles.map((r: { id: number }) => r.id),
    remark: d.remark || '',
  }
}

async function handleSubmit() {
  loading.value = true
  try {
    if (isEdit.value) {
      await updateAccount(accountId.value, {
        real_name: form.value.real_name,
        phone: form.value.phone,
        email: form.value.email,
        role_ids: form.value.role_ids,
        remark: form.value.remark,
      })
      ElMessage.success('已更新')
    } else {
      await createAccount({ ...form.value })
      ElMessage.success('已创建')
    }
    router.push('/admin/accounts')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: { message?: string; details?: { field: string; message: string }[] } } } }
    const detail = err?.response?.data?.error
    if (detail?.details) {
      ElMessage.error(detail.details.map((d: { message: string }) => d.message).join('; '))
    } else {
      ElMessage.error(detail?.message || '操作失败')
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => { loadRoles(); loadDetail() })
</script>

<template>
  <div class="account-form">
    <h2>{{ isEdit ? '编辑账号' : '创建账号' }}</h2>
    <form class="form-body" @submit.prevent="handleSubmit">
      <div class="field">
        <label>登录账号</label>
        <input v-model="form.username" :disabled="isEdit" placeholder="英文/数字" />
      </div>
      <div class="field" v-if="!isEdit">
        <label>初始密码</label>
        <input v-model="form.password" type="password" placeholder="至少 8 位" />
      </div>
      <div class="field">
        <label>姓名</label>
        <input v-model="form.real_name" placeholder="真实姓名" />
      </div>
      <div class="field">
        <label>手机号</label>
        <input v-model="form.phone" placeholder="可选" />
      </div>
      <div class="field">
        <label>邮箱</label>
        <input v-model="form.email" placeholder="可选" />
      </div>
      <div class="field">
        <label>角色</label>
        <div class="role-checks">
          <label v-for="r in roles" :key="r.id" class="role-check">
            <input type="checkbox" :value="r.id" v-model="form.role_ids" />
            {{ r.name }}
          </label>
        </div>
      </div>
      <div class="field">
        <label>备注</label>
        <input v-model="form.remark" placeholder="可选" />
      </div>
      <div class="form-actions">
        <button type="button" class="btn-cancel" @click="router.push('/admin/accounts')">取消</button>
        <button type="submit" class="btn-primary" :disabled="loading">{{ isEdit ? '保存' : '创建' }}</button>
      </div>
    </form>
  </div>
</template>

<style scoped>
h2 { color: #f7f8f8; margin-bottom: 20px; }
.form-body { max-width: 480px; }
.field { margin-bottom: 14px; }
.field label { display: block; color: #8a8f98; font-size: 13px; margin-bottom: 4px; }
.field input { width: 100%; padding: 8px 12px; background: #191a1b; border: 1px solid rgba(255,255,255,0.08); border-radius: 6px; color: #f7f8f8; font-size: 14px; outline: none; box-sizing: border-box; }
.field input:focus { border-color: #5e6ad2; }
.role-checks { display: flex; gap: 12px; }
.role-check { color: #d0d6e0; font-size: 13px; display: flex; align-items: center; gap: 4px; cursor: pointer; }
.form-actions { display: flex; gap: 8px; margin-top: 20px; }
.btn-primary { padding: 8px 20px; background: #5e6ad2; color: #fff; border: none; border-radius: 6px; cursor: pointer; }
.btn-cancel { padding: 8px 20px; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.08); color: #d0d6e0; border-radius: 6px; cursor: pointer; }
</style>
