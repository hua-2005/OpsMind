<script setup lang="ts">
// 角色权限管理页：角色列表 + 权限树编辑
import { ref, onMounted } from 'vue'
import { getRoles, getPermissionsTree, updateRolePermissions } from '@/api/roles'
import type { RoleItem, PermissionNode } from '@/api/roles'
import { ElMessage } from 'element-plus'

const roles = ref<RoleItem[]>([])
const permissions = ref<PermissionNode[]>([])
const selectedRole = ref<RoleItem | null>(null)
const checkedIds = ref<Set<number>>(new Set())
const loading = ref(false)

async function loadRoles() {
  const res = await getRoles({ per_page: 100 })
  roles.value = res.data.data
}

async function loadPermissions() {
  const res = await getPermissionsTree()
  permissions.value = res.data.data
}

function selectRole(role: RoleItem) {
  selectedRole.value = role
  checkedIds.value = new Set()
}

function togglePerm(id: number) {
  const s = new Set(checkedIds.value)
  if (s.has(id)) s.delete(id); else s.add(id)
  checkedIds.value = s
}

async function handleSave() {
  if (!selectedRole.value) return
  loading.value = true
  try {
    await updateRolePermissions(selectedRole.value.id, [...checkedIds.value])
    ElMessage.success('权限已更新')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="role-page">
    <h2>角色权限管理</h2>
    <div class="role-layout">
      <div class="role-list">
        <div
          v-for="r in roles" :key="r.id"
          class="role-item"
          :class="{ active: selectedRole?.id === r.id }"
          @click="selectRole(r)"
        >
          <div class="role-name">{{ r.name }}</div>
          <div class="role-code">{{ r.code }}</div>
        </div>
      </div>
      <div class="perm-panel" v-if="selectedRole">
        <div class="perm-header">
          <span>{{ selectedRole.name }} — 权限配置</span>
          <button class="btn-primary" :disabled="loading" @click="handleSave">保存</button>
        </div>
        <div class="perm-tree">
          <template v-for="p in permissions" :key="p.id">
            <label class="perm-item" style="padding-left:0">
              <input type="checkbox" :checked="checkedIds.has(p.id)" @change="togglePerm(p.id)" />
              <span class="perm-label">{{ p.name }}</span>
              <span class="perm-code">{{ p.code }}</span>
            </label>
            <label
              v-for="c in p.children" :key="c.id"
              class="perm-item"
              style="padding-left:20px"
            >
              <input type="checkbox" :checked="checkedIds.has(c.id)" @change="togglePerm(c.id)" />
              <span class="perm-label">{{ c.name }}</span>
              <span class="perm-code">{{ c.code }}</span>
            </label>
          </template>
        </div>
      </div>
      <div class="perm-panel" v-else>
        <p class="hint">请选择左侧角色以编辑权限</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
h2 { color: #f7f8f8; margin-bottom: 16px; }
.role-layout { display: flex; gap: 20px; }
.role-list { width: 200px; flex-shrink: 0; }
.role-item { padding: 10px 12px; border-radius: 6px; cursor: pointer; margin-bottom: 4px; border: 1px solid transparent; }
.role-item:hover { background: rgba(255,255,255,0.03); }
.role-item.active { background: rgba(94,106,210,0.12); border-color: rgba(94,106,210,0.3); }
.role-name { color: #d0d6e0; font-size: 13px; }
.role-code { color: #8a8f98; font-size: 12px; }
.perm-panel { flex: 1; background: #191a1b; border: 1px solid rgba(255,255,255,0.05); border-radius: 8px; padding: 16px; }
.perm-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; color: #d0d6e0; font-size: 14px; }
.perm-tree { max-height: 500px; overflow-y: auto; }
.perm-item { display: flex; align-items: center; gap: 6px; padding: 4px 0; cursor: pointer; font-size: 13px; }
.perm-label { color: #d0d6e0; }
.perm-code { color: #62666d; font-size: 11px; }
.hint { color: #8a8f98; }
.btn-primary { padding: 6px 14px; background: #5e6ad2; color: #fff; border: none; border-radius: 6px; cursor: pointer; font-size: 13px; }
</style>
