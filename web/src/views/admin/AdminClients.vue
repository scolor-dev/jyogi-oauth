<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../../api'

const clients = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const error = ref('')

onMounted(() => loadClients())

async function loadClients() {
  loading.value = true
  try {
    const data = await api.admin.listClients(page.value)
    clients.value = data.clients || []
    total.value = data.total
  } catch (e: any) {
    error.value = e.message || 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function deactivateClient(id: string) {
  if (!confirm('このクライアントを無効化しますか？')) return
  try {
    await api.admin.deleteAdminClient(id)
    await loadClients()
  } catch {}
}
</script>

<template>
  <div>
    <h1>All Clients</h1>
    <p class="subtitle">システム全体のOAuthクライアント一覧（admin のみ閲覧可能）</p>

    <div v-if="error" class="error-msg">{{ error }}</div>
    <div v-if="loading" class="loading">Loading...</div>

    <table v-if="!loading && clients.length" class="data-table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Client ID</th>
          <th>Type</th>
          <th>Redirect URIs</th>
          <th>Status</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="c in clients" :key="c.id" :class="{ inactive: !c.is_active }">
          <td>
            <strong>{{ c.name }}</strong>
            <div v-if="c.description" class="client-desc">{{ c.description }}</div>
          </td>
          <td><code>{{ c.client_id }}</code></td>
          <td><span class="type-badge" :class="'type-' + c.client_type">{{ c.client_type }}</span></td>
          <td>
            <div class="uri-list">
              <code v-for="uri in c.redirect_uris" :key="uri" class="uri-item">{{ uri }}</code>
            </div>
          </td>
          <td>
            <span :class="c.is_active ? 'status-active' : 'status-inactive'">
              {{ c.is_active ? 'Active' : 'Inactive' }}
            </span>
          </td>
          <td>{{ new Date(c.created_at).toLocaleDateString('ja-JP') }}</td>
          <td>
            <button v-if="c.is_active" class="btn-danger btn-sm" @click="deactivateClient(c.id)">
              Disable
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <p v-if="!loading && !clients.length" class="empty">No clients found</p>

    <div v-if="total > 20" class="pagination">
      <button class="btn-outline btn-sm" :disabled="page <= 1" @click="page--; loadClients()">Prev</button>
      <span>Page {{ page }}</span>
      <button class="btn-outline btn-sm" :disabled="clients.length < 20" @click="page++; loadClients()">Next</button>
    </div>
  </div>
</template>

<style scoped>
h1 { margin-bottom: 0.3rem; }
.subtitle { color: #888; font-size: 0.9rem; margin-bottom: 1.5rem; }
.data-table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.data-table th { background: #f5f5f5; text-align: left; padding: 0.7rem 0.8rem; font-size: 0.85rem; font-weight: 600; }
.data-table td { padding: 0.6rem 0.8rem; border-top: 1px solid #f0f0f0; font-size: 0.9rem; vertical-align: top; }
.inactive { opacity: 0.5; }
.client-desc { font-size: 0.8rem; color: #888; margin-top: 0.2rem; }
.type-badge { font-size: 0.7rem; padding: 0.15rem 0.5rem; border-radius: 3px; font-weight: 600; }
.type-public { background: #e8f5e9; color: #2e7d32; }
.type-confidential { background: #fff3e0; color: #e65100; }
.uri-list { display: flex; flex-direction: column; gap: 0.2rem; }
.uri-item { font-size: 0.8rem; }
.status-active { color: #2e7d32; font-weight: 500; }
.status-inactive { color: #c62828; font-weight: 500; }
.btn-sm { padding: 0.3rem 0.7rem; font-size: 0.8rem; }
.loading, .empty { color: #888; text-align: center; padding: 2rem; }
.pagination { display: flex; align-items: center; gap: 1rem; justify-content: center; margin-top: 1rem; font-size: 0.9rem; }
</style>
