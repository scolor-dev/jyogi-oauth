<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../../api'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const members = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const error = ref('')

const showCreate = ref(false)
const newMember = ref({ username: '', password: '', email: '', must_change_password: true })
const createError = ref('')

const editingId = ref<string | null>(null)
const editRole = ref('')
const tempPassword = ref('')
const tempPasswordFor = ref('')

onMounted(() => loadMembers())

async function loadMembers() {
  loading.value = true
  try {
    const data = await api.admin.listMembers(page.value)
    members.value = data.members || []
    total.value = data.total
  } catch (e: any) {
    error.value = e.message || 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function createMember() {
  createError.value = ''
  if (!newMember.value.username || !newMember.value.email) {
    createError.value = 'Username and email are required'
    return
  }
  try {
    const payload: any = {
      username: newMember.value.username,
      email: newMember.value.email,
      must_change_password: newMember.value.must_change_password,
    }
    if (newMember.value.password) {
      payload.password = newMember.value.password
    }
    const res = await api.admin.createMember(payload)
    if (res.temporary_password) {
      tempPassword.value = res.temporary_password
      tempPasswordFor.value = newMember.value.username
    }
    newMember.value = { username: '', password: '', email: '', must_change_password: true }
    showCreate.value = false
    await loadMembers()
  } catch (e: any) {
    createError.value = e.message || 'Failed to create'
  }
}

function startEditRole(member: any) {
  editingId.value = member.id
  editRole.value = member.role
}

async function saveRole(id: string) {
  try {
    await api.admin.updateMember(id, { role: editRole.value })
    editingId.value = null
    await loadMembers()
  } catch {}
}

async function resetPassword(member: any) {
  if (!confirm(`${member.username} のパスワードをリセットしますか？`)) return
  tempPassword.value = ''
  try {
    const res = await api.admin.resetPassword(member.id)
    tempPassword.value = res.temporary_password
    tempPasswordFor.value = member.username
  } catch {}
}

async function toggleActive(member: any) {
  const action = member.is_active ? 'このメンバーを無効化しますか？' : 'このメンバーを有効化しますか？'
  if (!confirm(action)) return
  try {
    await api.admin.updateMember(member.id, { is_active: !member.is_active })
    await loadMembers()
  } catch {}
}

const roleLevels: Record<string, number> = { admin: 3, moderator: 2, member: 1 }

function canManage(target: any): boolean {
  const me = auth.member
  if (!me || me.id === target.id) return false
  if (target.username === 'root') return false
  return (roleLevels[me.role] || 0) > (roleLevels[target.role] || 0)
}

function canEditRole(target: any): boolean {
  const me = auth.member
  if (!me || me.role !== 'admin') return false
  return canManage(target)
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Members</h1>
      <button class="btn-primary btn-sm" @click="showCreate = !showCreate">
        {{ showCreate ? 'Cancel' : '+ New Member' }}
      </button>
    </div>

    <div v-if="showCreate" class="card create-form">
      <h3>Create Member</h3>
      <div class="form-row">
        <div class="form-group">
          <label>Username</label>
          <input v-model="newMember.username">
        </div>
        <div class="form-group">
          <label>Email</label>
          <input v-model="newMember.email" type="email">
        </div>
        <div class="form-group">
          <label>Password <span class="hint">(空欄で自動生成)</span></label>
          <input v-model="newMember.password" type="password" placeholder="自動生成">
        </div>
      </div>
      <label class="checkbox-label">
        <input type="checkbox" v-model="newMember.must_change_password">
        初回ログイン時にパスワード変更を強制する
      </label>
      <p v-if="createError" class="error-msg">{{ createError }}</p>
      <button class="btn-primary" @click="createMember">Create</button>
    </div>

    <div v-if="error" class="error-msg">{{ error }}</div>
    <div v-if="loading" class="loading">Loading...</div>

    <div v-if="tempPassword" class="temp-password-alert">
      <p><strong>{{ tempPasswordFor }}</strong> の一時パスワード（一度だけ表示）:</p>
      <code class="temp-password-code">{{ tempPassword }}</code>
      <button class="btn-sm btn-outline" @click="tempPassword = ''">閉じる</button>
    </div>

    <table v-if="!loading && members.length" class="data-table">
      <thead>
        <tr>
          <th>Username</th>
          <th>Email</th>
          <th>Role</th>
          <th>Status</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="m in members" :key="m.id" :class="{ inactive: !m.is_active }">
          <td><strong>{{ m.username }}</strong></td>
          <td>{{ m.email }}</td>
          <td>
            <template v-if="editingId === m.id && canEditRole(m)">
              <select v-model="editRole" class="role-select">
                <option value="member">member</option>
                <option value="moderator">moderator</option>
                <option value="admin">admin</option>
              </select>
              <button class="btn-sm btn-primary" @click="saveRole(m.id)">Save</button>
              <button class="btn-sm btn-outline" @click="editingId = null">Cancel</button>
            </template>
            <template v-else>
              <span class="role-badge" :class="'role-' + m.role">{{ m.role }}</span>
              <button v-if="canEditRole(m)"
                class="btn-sm btn-outline edit-btn" @click="startEditRole(m)">Edit</button>
            </template>
          </td>
          <td>
            <span :class="m.is_active ? 'status-active' : 'status-inactive'">
              {{ m.is_active ? 'Active' : 'Inactive' }}
            </span>
          </td>
          <td>{{ new Date(m.created_at).toLocaleDateString('ja-JP') }}</td>
          <td class="actions-cell">
            <button v-if="canManage(m)" class="btn-sm btn-outline"
              @click="resetPassword(m)">Reset PW</button>
            <button v-if="canManage(m)" class="btn-sm"
              :class="m.is_active ? 'btn-danger' : 'btn-primary'"
              @click="toggleActive(m)">
              {{ m.is_active ? 'Disable' : 'Enable' }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <p v-if="!loading && !members.length" class="empty">No members found</p>

    <div v-if="total > 20" class="pagination">
      <button class="btn-outline btn-sm" :disabled="page <= 1" @click="page--; loadMembers()">Prev</button>
      <span>Page {{ page }}</span>
      <button class="btn-outline btn-sm" :disabled="members.length < 20" @click="page++; loadMembers()">Next</button>
    </div>
  </div>
</template>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
h1 { margin: 0; }
.create-form { margin-bottom: 1.5rem; }
.create-form h3 { margin-bottom: 1rem; }
.form-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.8rem; margin-bottom: 0.8rem; }
.data-table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.data-table th { background: #f5f5f5; text-align: left; padding: 0.7rem 0.8rem; font-size: 0.85rem; font-weight: 600; }
.data-table td { padding: 0.6rem 0.8rem; border-top: 1px solid #f0f0f0; font-size: 0.9rem; }
.inactive { opacity: 0.5; }
.role-badge { font-size: 0.7rem; padding: 0.15rem 0.5rem; border-radius: 3px; font-weight: 600; text-transform: uppercase; }
.role-admin { background: #fce4ec; color: #c62828; }
.role-moderator { background: #fff3e0; color: #e65100; }
.role-member { background: #e8f5e9; color: #2e7d32; }
.role-select { width: auto; padding: 0.2rem 0.4rem; font-size: 0.8rem; margin-right: 0.3rem; }
.edit-btn { margin-left: 0.4rem; padding: 0.1rem 0.4rem; font-size: 0.7rem; }
.status-active { color: #2e7d32; font-weight: 500; }
.status-inactive { color: #c62828; font-weight: 500; }
.btn-sm { padding: 0.3rem 0.7rem; font-size: 0.8rem; }
.loading { color: #888; padding: 2rem; text-align: center; }
.empty { color: #888; text-align: center; padding: 2rem; }
.pagination { display: flex; align-items: center; gap: 1rem; justify-content: center; margin-top: 1rem; font-size: 0.9rem; }
.actions-cell { display: flex; gap: 0.4rem; }
.temp-password-alert {
  background: #fff3cd; border: 1px solid #ffc107; border-radius: 6px;
  padding: 1rem; margin-bottom: 1rem; display: flex; align-items: center; gap: 1rem; flex-wrap: wrap;
}
.temp-password-code {
  display: inline-block; padding: 0.4rem 0.8rem; background: #fff; border: 1px solid #ddd;
  border-radius: 4px; font-size: 1rem; font-weight: 600; letter-spacing: 0.05em;
}
.checkbox-label { display: flex; align-items: center; gap: 0.4rem; font-size: 0.9rem; cursor: pointer; margin-bottom: 0.8rem; }
.checkbox-label input[type="checkbox"] { cursor: pointer; }
.hint { font-weight: 400; color: #aaa; font-size: 0.8rem; }
</style>
