<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../../api'
import { getErrorMessage, type AuditLog } from '../../types'

const logs = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const perPage = 20
const loading = ref(true)
const error = ref('')
const filters = ref({ action: '', memberId: '', from: '', to: '' })

onMounted(() => loadLogs())

function toRFC3339(value: string): string | undefined {
  if (!value) return undefined
  return new Date(value).toISOString()
}

async function loadLogs(resetPage = false) {
  if (resetPage) page.value = 1
  loading.value = true
  error.value = ''
  try {
    const data = await api.admin.listAuditLogs({
      page: page.value,
      perPage,
      action: filters.value.action.trim() || undefined,
      memberId: filters.value.memberId.trim() || undefined,
      from: toRFC3339(filters.value.from),
      to: toRFC3339(filters.value.to),
    })
    logs.value = data.items || []
    total.value = data.total
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to load audit logs')
  } finally {
    loading.value = false
  }
}

function formatDetails(details: Record<string, unknown> | null): string {
  if (!details) return '-'
  return JSON.stringify(details)
}
</script>

<template>
  <div>
    <h1>Audit Logs</h1>

    <div class="filters card">
      <div class="form-group">
        <label>Action</label>
        <input v-model="filters.action" placeholder="login_success">
      </div>
      <div class="form-group">
        <label>Member ID</label>
        <input v-model="filters.memberId" placeholder="UUID">
      </div>
      <div class="form-group">
        <label>From</label>
        <input v-model="filters.from" type="datetime-local">
      </div>
      <div class="form-group">
        <label>To</label>
        <input v-model="filters.to" type="datetime-local">
      </div>
      <button class="btn-primary btn-sm" @click="loadLogs(true)">Filter</button>
    </div>

    <div v-if="error" class="error-msg">{{ error }}</div>
    <div v-if="loading" class="loading">Loading...</div>

    <table v-if="!loading && logs.length" class="data-table">
      <thead>
        <tr>
          <th>Created</th>
          <th>Action</th>
          <th>Member</th>
          <th>Client</th>
          <th>IP</th>
          <th>Details</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="log in logs" :key="log.id">
          <td>{{ new Date(log.created_at).toLocaleString('ja-JP') }}</td>
          <td><code>{{ log.action }}</code></td>
          <td><code v-if="log.member_id">{{ log.member_id }}</code><span v-else>-</span></td>
          <td><code v-if="log.client_id">{{ log.client_id }}</code><span v-else>-</span></td>
          <td>{{ log.ip_address || '-' }}</td>
          <td><code class="details">{{ formatDetails(log.details) }}</code></td>
        </tr>
      </tbody>
    </table>

    <p v-if="!loading && !logs.length" class="empty">No audit logs found</p>

    <div v-if="total > perPage" class="pagination">
      <button class="btn-outline btn-sm" :disabled="page <= 1" @click="page--; loadLogs()">Prev</button>
      <span>Page {{ page }}</span>
      <button class="btn-outline btn-sm" :disabled="logs.length < perPage" @click="page++; loadLogs()">Next</button>
    </div>
  </div>
</template>

<style scoped>
h1 { margin-bottom: 1.5rem; }
.filters {
  display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)) auto;
  gap: 0.8rem; align-items: end; margin-bottom: 1rem;
}
.data-table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.data-table th { background: #f5f5f5; text-align: left; padding: 0.7rem 0.8rem; font-size: 0.85rem; font-weight: 600; }
.data-table td { padding: 0.6rem 0.8rem; border-top: 1px solid #f0f0f0; font-size: 0.85rem; vertical-align: top; }
.details { display: block; max-width: 260px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.btn-sm { padding: 0.3rem 0.7rem; font-size: 0.8rem; }
.loading, .empty { color: #888; text-align: center; padding: 2rem; }
.pagination { display: flex; align-items: center; gap: 1rem; justify-content: center; margin-top: 1rem; font-size: 0.9rem; }
@media (max-width: 700px) {
  .filters { grid-template-columns: 1fr; }
}
</style>
