<script setup lang="ts">
// 账号列表页：分页表格、筛选、冻结/恢复操作
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getAccounts, freezeAccount, restoreAccount } from '@/api/accounts'
import type { AccountItem } from '@/api/accounts'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const list = ref<AccountItem[]>([])
const total = ref(0)
const page = ref(1)
const perPage = ref(20)
const loading = ref(false)
const q = ref('')
const status = ref('')

async function fetchList() {
  loading.value = true
  try {
    const res = await getAccounts({ page: page.value, per_page: perPage.value, q: q.value, status: status.value })
    list.value = res.data.data
    total.value = res.data.meta.total
  } finally {
    loading.value = false
  }
}

async function handleFreeze(row: AccountItem) {
  try {
    await ElMessageBox.prompt('请输入冻结原因', '冻结账号', { confirmButtonText: '确认', cancelButtonText: '取消' })
      .then(async ({ value }) => {
        await freezeAccount(row.id, value)
        ElMessage.success('已冻结')
        fetchList()
      })
  } catch { /* cancelled */ }
}

async function handleRestore(row: AccountItem) {
  try {
    await ElMessageBox.prompt('请输入恢复原因', '恢复账号', { confirmButtonText: '确认', cancelButtonText: '取消' })
      .then(async ({ value }) => {
        await restoreAccount(row.id, value)
        ElMessage.success('已恢复')
        fetchList()
      })
  } catch { /* cancelled */ }
}

function statusTag(s: string) {
  return s === 'active' ? 'success' : 'danger'
}

onMounted(fetchList)
</script>

<template>
  <div class="account-list">
    <div class="page-header">
      <h2>账号管理</h2>
      <button class="btn-primary" @click="router.push('/admin/accounts/create')">创建账号</button>
    </div>
    <div class="filters">
      <input v-model="q" placeholder="搜索账号/姓名/手机号" @keyup.enter="fetchList" />
      <select v-model="status" @change="fetchList">
        <option value="">全部状态</option>
        <option value="active">正常</option>
        <option value="frozen">冻结</option>
      </select>
      <button class="btn-search" @click="fetchList">搜索</button>
    </div>
    <table v-loading="loading">
      <thead>
        <tr>
          <th>ID</th><th>账号</th><th>姓名</th><th>状态</th><th>最近登录</th><th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in list" :key="row.id">
          <td>{{ row.id }}</td>
          <td>{{ row.username }}</td>
          <td>{{ row.real_name }}</td>
          <td><span :class="'tag tag-' + statusTag(row.status)">{{ row.status === 'active' ? '正常' : '冻结' }}</span></td>
          <td>{{ row.last_login_at || '-' }}</td>
          <td>
            <button class="btn-link" @click="router.push('/admin/accounts/' + row.id)">编辑</button>
            <button v-if="row.status === 'active'" class="btn-link warn" @click="handleFreeze(row)">冻结</button>
            <button v-if="row.status === 'frozen'" class="btn-link" @click="handleRestore(row)">恢复</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div class="pager">
      <span>共 {{ total }} 条</span>
    </div>
  </div>
</template>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h2 { color: #f7f8f8; font-size: 20px; margin: 0; }
.filters { display: flex; gap: 8px; margin-bottom: 16px; }
.filters input, .filters select { padding: 8px 12px; background: #191a1b; border: 1px solid rgba(255,255,255,0.08); border-radius: 6px; color: #f7f8f8; font-size: 13px; outline: none; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: 10px 12px; text-align: left; border-bottom: 1px solid rgba(255,255,255,0.05); color: #d0d6e0; font-size: 13px; }
th { color: #8a8f98; font-weight: 510; }
.btn-primary { padding: 8px 16px; background: #5e6ad2; color: #fff; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; }
.btn-search { padding: 8px 14px; background: rgba(255,255,255,0.06); border: 1px solid rgba(255,255,255,0.08); color: #d0d6e0; border-radius: 6px; cursor: pointer; font-size: 13px; }
.btn-link { background: none; border: none; color: #7170ff; cursor: pointer; font-size: 13px; padding: 0 6px; }
.btn-link.warn { color: #e55c5c; }
.tag { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 12px; }
.tag-success { background: rgba(39,166,68,0.15); color: #27a644; }
.tag-danger { background: rgba(229,92,92,0.15); color: #e55c5c; }
.pager { margin-top: 16px; color: #8a8f98; font-size: 13px; }
</style>
