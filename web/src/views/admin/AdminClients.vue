<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../../api'
import { getErrorMessage, type OAuthClient } from '../../types'

const clients = ref<OAuthClient[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const error = ref('')
const rotatedSecret = ref<{ clientId: string; secret: string } | null>(null)

onMounted(() => loadClients())

async function loadClients() {
  loading.value = true
  try {
    const data = await api.admin.listClients(page.value)
    clients.value = data.clients || []
    total.value = data.total
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to load')
  } finally {
    loading.value = false
  }
}

async function deactivateClient(id: string) {
  if (!confirm('このクライアントを無効化しますか？')) return
  try {
    await api.admin.deleteAdminClient(id)
    await loadClients()
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to disable client')
  }
}

async function rotateClientSecret(client: OAuthClient) {
  if (!confirm(`${client.name} のclient secretを再発行しますか？既存のsecretは使えなくなります。`)) return
  try {
    const res = await api.admin.rotateClientSecret(client.id)
    rotatedSecret.value = { clientId: client.id, secret: res.client_secret }
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to rotate client secret')
  }
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
            <div class="client-cell">
              <div class="client-icon" aria-hidden="true">
                <img v-if="c.icon_url" :src="c.icon_url" alt="">
                <span v-else>{{ c.name.charAt(0).toUpperCase() }}</span>
              </div>
              <div>
                <strong>{{ c.name }}</strong>
                <div v-if="c.description" class="client-desc">{{ c.description }}</div>
                <div v-if="rotatedSecret?.clientId === c.id" class="secret-display">
                  <p><strong>New secret:</strong></p>
                  <code>{{ rotatedSecret.secret }}</code>
                </div>
              </div>
            </div>
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
            <button v-if="c.is_active && c.client_type === 'confidential'" class="btn-outline btn-sm" @click="rotateClientSecret(c)">
              Rotate
            </button>
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
.client-cell { display: flex; gap: 0.65rem; align-items: flex-start; }
.client-icon {
  width: 38px; height: 38px; border-radius: 8px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  background: #eef2f7; color: #345; font-weight: 700; overflow: hidden;
}
.client-icon img { width: 100%; height: 100%; object-fit: cover; }
.client-desc { font-size: 0.8rem; color: #888; margin-top: 0.2rem; }
.secret-display {
  margin-top: 0.5rem; padding: 0.5rem; background: #fff8e1;
  border: 1px solid #ffe082; border-radius: 6px; font-size: 0.8rem;
}
.secret-display code { display: block; word-break: break-all; margin-top: 0.25rem; }
.type-badge { font-size: 0.7rem; padding: 0.15rem 0.5rem; border-radius: 3px; font-weight: 600; }
.type-public { background: #e8f5e9; color: #2e7d32; }
.type-confidential { background: #fff3e0; color: #e65100; }
.uri-list { display: flex; flex-direction: column; gap: 0.2rem; }
.uri-item { font-size: 0.8rem; }
.status-active { color: #2e7d32; font-weight: 500; }
.status-inactive { color: #c62828; font-weight: 500; }
.btn-sm { padding: 0.3rem 0.7rem; font-size: 0.8rem; margin-right: 0.3rem; }
.loading, .empty { color: #888; text-align: center; padding: 2rem; }
.pagination { display: flex; align-items: center; gap: 1rem; justify-content: center; margin-top: 1rem; font-size: 0.9rem; }
</style>
