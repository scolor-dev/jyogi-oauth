<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

const clientName = ref('')
const scopes = ref<{ name: string; description?: string }[]>([])
const selectedScopes = ref<string[]>([])
const error = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    const data = await api.getConsentInfo()
    clientName.value = data.client_name
    scopes.value = data.requested_scopes || []
    selectedScopes.value = scopes.value.map(s => s.name)
  } catch (e: any) {
    error.value = e.message || 'Failed to load consent info'
  } finally {
    loading.value = false
  }
})

async function handleConsent(approved: boolean) {
  loading.value = true
  try {
    const res = await api.submitConsent(approved, selectedScopes.value)
    if (res.redirect_to) {
      window.location.href = res.redirect_to
    }
  } catch (e: any) {
    error.value = e.message || 'Failed to process consent'
    loading.value = false
  }
}
</script>

<template>
  <div class="container consent-page">
    <div class="card consent-card">
      <h2>Authorization Request</h2>

      <div v-if="loading" class="loading">Loading...</div>

      <div v-else-if="error" class="error-msg">{{ error }}</div>

      <template v-else>
        <p class="consent-desc">
          <strong>{{ clientName }}</strong> があなたのアカウントへのアクセスを要求しています。
        </p>

        <div class="scopes-list">
          <h3>要求されている権限</h3>
          <div v-for="scope in scopes" :key="scope.name" class="scope-item">
            <label>
              <input type="checkbox" :value="scope.name" v-model="selectedScopes" checked>
              <span class="scope-name">{{ scope.name }}</span>
              <span class="scope-desc" v-if="scope.description">{{ scope.description }}</span>
            </label>
          </div>
        </div>

        <div class="consent-actions">
          <button class="btn-primary" @click="handleConsent(true)" :disabled="loading">許可する</button>
          <button class="btn-outline" @click="handleConsent(false)" :disabled="loading">拒否する</button>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.consent-page { display: flex; justify-content: center; padding-top: 3rem; }
.consent-card { max-width: 480px; width: 100%; }
.consent-card h2 { margin-bottom: 1rem; }
.consent-desc { margin-bottom: 1.5rem; font-size: 0.95rem; color: #555; }
.scopes-list { margin-bottom: 1.5rem; }
.scopes-list h3 { font-size: 0.95rem; margin-bottom: 0.6rem; }
.scope-item { padding: 0.5rem 0; border-bottom: 1px solid #f0f0f0; }
.scope-item label { display: flex; align-items: center; gap: 0.5rem; cursor: pointer; }
.scope-name { font-weight: 500; font-size: 0.9rem; }
.scope-desc { color: #888; font-size: 0.8rem; }
.consent-actions { display: flex; gap: 0.8rem; }
.consent-actions button { flex: 1; }
.loading { text-align: center; color: #888; padding: 2rem; }
</style>
