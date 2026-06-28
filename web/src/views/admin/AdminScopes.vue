<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../../api'
import { getErrorMessage, type Scope } from '../../types'

const scopes = ref<Scope[]>([])
const loading = ref(true)
const error = ref('')
const showCreate = ref(false)
const createError = ref('')
const newScope = ref({ name: '', description: '', is_default: false })
const editingId = ref<string | null>(null)
const editScope = ref({ name: '', description: '', is_default: false })

onMounted(() => loadScopes())

async function loadScopes() {
  loading.value = true
  try {
    const data = await api.admin.listScopes()
    scopes.value = data.scopes || []
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to load scopes')
  } finally {
    loading.value = false
  }
}

async function createScope() {
  createError.value = ''
  if (!newScope.value.name.trim()) {
    createError.value = 'Name is required'
    return
  }
  try {
    await api.admin.createScope({
      name: newScope.value.name.trim(),
      description: newScope.value.description.trim() || undefined,
      is_default: newScope.value.is_default,
    })
    newScope.value = { name: '', description: '', is_default: false }
    showCreate.value = false
    await loadScopes()
  } catch (e: unknown) {
    createError.value = getErrorMessage(e, 'Failed to create scope')
  }
}

function startEdit(scope: Scope) {
  editingId.value = scope.id
  editScope.value = {
    name: scope.name,
    description: scope.description || '',
    is_default: scope.is_default,
  }
}

async function saveScope(id: string) {
  if (!editScope.value.name.trim()) {
    error.value = 'Name is required'
    return
  }
  try {
    await api.admin.updateScope(id, {
      name: editScope.value.name.trim(),
      description: editScope.value.description.trim(),
      is_default: editScope.value.is_default,
    })
    editingId.value = null
    await loadScopes()
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to update scope')
  }
}

async function deleteScope(scope: Scope) {
  if (!confirm(`${scope.name} を削除しますか？`)) return
  try {
    await api.admin.deleteScope(scope.id)
    await loadScopes()
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to delete scope')
  }
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1>Scopes</h1>
      <button class="btn-primary btn-sm" @click="showCreate = !showCreate">
        {{ showCreate ? 'Cancel' : '+ New Scope' }}
      </button>
    </div>

    <div v-if="showCreate" class="card create-form">
      <h3>Create Scope</h3>
      <div class="form-row">
        <div class="form-group">
          <label>Name</label>
          <input v-model="newScope.name" placeholder="profile">
        </div>
        <div class="form-group">
          <label>Description</label>
          <input v-model="newScope.description" placeholder="Display name and profile information">
        </div>
      </div>
      <label class="checkbox-label">
        <input type="checkbox" v-model="newScope.is_default">
        Default scope
      </label>
      <p v-if="createError" class="error-msg">{{ createError }}</p>
      <button class="btn-primary" @click="createScope">Create</button>
    </div>

    <div v-if="error" class="error-msg">{{ error }}</div>
    <div v-if="loading" class="loading">Loading...</div>

    <table v-if="!loading && scopes.length" class="data-table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Description</th>
          <th>Default</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="scope in scopes" :key="scope.id">
          <template v-if="editingId === scope.id">
            <td><input v-model="editScope.name" class="table-input"></td>
            <td><input v-model="editScope.description" class="table-input"></td>
            <td>
              <label class="inline-check">
                <input type="checkbox" v-model="editScope.is_default">
                Default
              </label>
            </td>
            <td>{{ new Date(scope.created_at).toLocaleDateString('ja-JP') }}</td>
            <td class="actions-cell">
              <button class="btn-primary btn-sm" @click="saveScope(scope.id)">Save</button>
              <button class="btn-outline btn-sm" @click="editingId = null">Cancel</button>
            </td>
          </template>
          <template v-else>
            <td><code>{{ scope.name }}</code></td>
            <td>{{ scope.description || '-' }}</td>
            <td>
              <span :class="scope.is_default ? 'status-active' : 'status-muted'">
                {{ scope.is_default ? 'Yes' : 'No' }}
              </span>
            </td>
            <td>{{ new Date(scope.created_at).toLocaleDateString('ja-JP') }}</td>
            <td class="actions-cell">
              <button class="btn-outline btn-sm" @click="startEdit(scope)">Edit</button>
              <button class="btn-danger btn-sm" @click="deleteScope(scope)">Delete</button>
            </td>
          </template>
        </tr>
      </tbody>
    </table>

    <p v-if="!loading && !scopes.length" class="empty">No scopes found</p>
  </div>
</template>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
h1 { margin: 0; }
.create-form { margin-bottom: 1.5rem; }
.create-form h3 { margin-bottom: 1rem; }
.form-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 0.8rem; margin-bottom: 0.8rem; }
.data-table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.data-table th { background: #f5f5f5; text-align: left; padding: 0.7rem 0.8rem; font-size: 0.85rem; font-weight: 600; }
.data-table td { padding: 0.6rem 0.8rem; border-top: 1px solid #f0f0f0; font-size: 0.9rem; vertical-align: middle; }
.table-input { padding: 0.4rem 0.5rem; font-size: 0.85rem; }
.status-active { color: #2e7d32; font-weight: 600; }
.status-muted { color: #888; }
.btn-sm { padding: 0.3rem 0.7rem; font-size: 0.8rem; }
.actions-cell { display: flex; gap: 0.4rem; }
.loading, .empty { color: #888; text-align: center; padding: 2rem; }
.checkbox-label, .inline-check { display: flex; align-items: center; gap: 0.4rem; font-size: 0.9rem; cursor: pointer; }
.checkbox-label { margin-bottom: 0.8rem; }
</style>
