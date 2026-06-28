<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const tab = ref<'profile' | 'apps' | 'clients' | 'sessions'>('profile')
const consents = ref<any[]>([])
const myClients = ref<any[]>([])
const sessions = ref<any[]>([])
const sessionsLoaded = ref(false)
const saving = ref(false)
const saveMsg = ref('')

const displayName = ref('')
const avatarUrl = ref('')
const themeColor = ref('#000000')
const tagline = ref('')

// Client creation
const showCreateClient = ref(false)
const newClient = ref({ name: '', client_type: 'public', redirect_uris: '', description: '', grant_types: ['authorization_code'] as string[] })
const createdSecret = ref('')
const clientError = ref('')

// Client editing
const editingClientId = ref<string | null>(null)
const editClient = ref({ name: '', redirect_uris: '', grant_types: [] as string[] })
const editError = ref('')

onMounted(async () => {
  if (!auth.loaded) await auth.fetchMe()
  if (!auth.isLoggedIn()) {
    router.push('/login')
    return
  }
  if (auth.identity) {
    displayName.value = auth.identity.display_name
    avatarUrl.value = auth.identity.avatar_url || ''
    themeColor.value = auth.identity.theme_color || '#000000'
    tagline.value = auth.identity.tagline || ''
  } else {
    displayName.value = auth.member?.username || ''
  }
  loadConsents()
  loadClients()
})

watch(tab, (val) => {
  if (val === 'sessions' && !sessionsLoaded.value) {
    loadSessions()
  }
})

async function loadConsents() {
  try {
    const data = await api.getConsents()
    consents.value = data.consents || []
  } catch {}
}

async function loadClients() {
  try {
    const data = await api.getMyClients()
    myClients.value = data.clients || []
  } catch {}
}

async function saveProfile() {
  saving.value = true
  saveMsg.value = ''
  try {
    await api.updateIdentity({
      display_name: displayName.value,
      avatar_url: avatarUrl.value || null,
      theme_color: themeColor.value,
      tagline: tagline.value || null,
    })
    await auth.fetchMe()
    saveMsg.value = 'Saved'
    setTimeout(() => saveMsg.value = '', 2000)
  } catch (e: any) {
    saveMsg.value = e.message || 'Failed to save'
  } finally {
    saving.value = false
  }
}

async function loadSessions() {
  try {
    const data = await api.getSessions()
    sessions.value = data.sessions || []
    sessionsLoaded.value = true
  } catch {}
}

function parseUA(ua: string): string {
  if (!ua) return 'Unknown'
  if (ua.includes('Firefox')) return 'Firefox'
  if (ua.includes('Edg/')) return 'Edge'
  if (ua.includes('Chrome')) return 'Chrome'
  if (ua.includes('Safari')) return 'Safari'
  return ua.substring(0, 30)
}

async function revokeSession(sessionId: string) {
  if (!confirm('このセッションを無効化しますか？')) return
  try {
    await api.revokeSession(sessionId)
    await loadSessions()
  } catch {}
}

async function revokeApp(clientId: string) {
  if (!confirm('このアプリのアクセスを取り消しますか？')) return
  try {
    await api.revokeConsent(clientId)
    await loadConsents()
  } catch {}
}

async function createClient() {
  clientError.value = ''
  createdSecret.value = ''
  const uris = newClient.value.redirect_uris.split('\n').map(u => u.trim()).filter(Boolean)
  if (!newClient.value.name || uris.length === 0) {
    clientError.value = 'Name and at least one redirect URI are required'
    return
  }
  try {
    const res = await api.createClient({
      name: newClient.value.name,
      client_type: newClient.value.client_type,
      redirect_uris: uris,
      allowed_grant_types: newClient.value.grant_types,
      description: newClient.value.description || undefined,
    })
    if (res.client_secret) {
      createdSecret.value = res.client_secret
    }
    newClient.value = { name: '', client_type: 'public', redirect_uris: '', description: '', grant_types: ['authorization_code'] }
    await loadClients()
    if (!res.client_secret) showCreateClient.value = false
  } catch (e: any) {
    clientError.value = e.message || 'Failed to create client'
  }
}

function startEditClient(client: any) {
  editingClientId.value = client.id
  editClient.value = {
    name: client.name,
    redirect_uris: client.redirect_uris.join('\n'),
    grant_types: [...client.allowed_grant_types],
  }
  editError.value = ''
}

function cancelEdit() {
  editingClientId.value = null
  editError.value = ''
}

async function saveEditClient(id: string) {
  editError.value = ''
  const uris = editClient.value.redirect_uris.split('\n').map(u => u.trim()).filter(Boolean)
  if (!editClient.value.name || uris.length === 0) {
    editError.value = 'Name and at least one redirect URI are required'
    return
  }
  if (editClient.value.grant_types.length === 0) {
    editError.value = 'At least one grant type is required'
    return
  }
  try {
    await api.updateClient(id, {
      name: editClient.value.name,
      redirect_uris: uris,
      allowed_grant_types: editClient.value.grant_types,
    })
    editingClientId.value = null
    await loadClients()
  } catch (e: any) {
    editError.value = e.message || 'Failed to update client'
  }
}

async function deleteClient(id: string) {
  if (!confirm('このクライアントを削除しますか？')) return
  try {
    await api.deleteClient(id)
    await loadClients()
  } catch {}
}

const previewStyle = computed(() => ({
  borderColor: themeColor.value,
  color: themeColor.value,
}))
</script>

<template>
  <div class="container dashboard" v-if="auth.isLoggedIn()">
    <h1>Dashboard</h1>
    <p class="welcome">Welcome, <strong>{{ auth.member?.username }}</strong>
      <span class="role-badge" :class="'role-' + auth.member?.role">{{ auth.member?.role }}</span>
    </p>

    <div class="tabs">
      <button :class="['tab', { active: tab === 'profile' }]" @click="tab = 'profile'">Profile</button>
      <button :class="['tab', { active: tab === 'apps' }]" @click="tab = 'apps'">Connected Apps</button>
      <button :class="['tab', { active: tab === 'clients' }]" @click="tab = 'clients'">My Clients</button>
      <button :class="['tab', { active: tab === 'sessions' }]" @click="tab = 'sessions'">Sessions</button>
    </div>

    <!-- Profile Tab -->
    <div v-if="tab === 'profile'" class="card">
      <h2>Profile</h2>
      <div class="profile-layout">
        <div class="profile-form">
          <div class="form-group">
            <label for="displayName">Display Name</label>
            <input id="displayName" v-model="displayName" maxlength="100">
          </div>
          <div class="form-group">
            <label for="avatarUrl">Avatar URL</label>
            <input id="avatarUrl" v-model="avatarUrl" placeholder="https://...">
          </div>
          <div class="form-group">
            <label for="tagline">Tagline <span class="hint">(max 8 chars)</span></label>
            <input id="tagline" v-model="tagline" maxlength="8" placeholder="e.g. Engineer">
          </div>
          <div class="form-group">
            <label for="themeColor">Theme Color</label>
            <div class="color-input">
              <input id="themeColor" type="color" v-model="themeColor">
              <span class="color-hex">{{ themeColor }}</span>
            </div>
          </div>
          <div class="form-actions">
            <button class="btn-primary" @click="saveProfile" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
            <span v-if="saveMsg" class="save-msg">{{ saveMsg }}</span>
          </div>
        </div>
        <div class="profile-preview">
          <div class="preview-card" :style="previewStyle">
            <div class="preview-avatar">
              <img v-if="avatarUrl" :src="avatarUrl" alt="avatar" @error="($event.target as HTMLImageElement).style.display='none'">
              <div v-else class="avatar-placeholder" :style="{ background: themeColor }">
                {{ displayName?.charAt(0)?.toUpperCase() || '?' }}
              </div>
            </div>
            <div class="preview-name">{{ displayName || 'No name' }}</div>
            <div v-if="tagline" class="preview-tagline">{{ tagline }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Connected Apps Tab -->
    <div v-if="tab === 'apps'" class="card">
      <h2>Connected Apps</h2>
      <p v-if="!consents.length" class="empty">No connected apps</p>
      <div v-for="consent in consents" :key="consent.client_id" class="app-item">
        <div class="app-info">
          <div class="app-name">{{ consent.client_name }}</div>
          <div class="app-scopes">
            <span v-for="scope in consent.scopes" :key="scope" class="scope-badge">{{ scope }}</span>
          </div>
          <div class="app-date">Granted: {{ new Date(consent.granted_at).toLocaleDateString('ja-JP') }}</div>
        </div>
        <button class="btn-danger btn-sm" @click="revokeApp(consent.client_id)">Revoke</button>
      </div>
    </div>

    <!-- My Clients Tab -->
    <div v-if="tab === 'clients'" class="card">
      <div class="section-header">
        <h2>My Clients</h2>
        <button class="btn-primary btn-sm" @click="showCreateClient = !showCreateClient">
          {{ showCreateClient ? 'Cancel' : '+ New Client' }}
        </button>
      </div>

      <!-- Create Client Form -->
      <div v-if="showCreateClient" class="create-form">
        <div class="form-group">
          <label>Application Name</label>
          <input v-model="newClient.name" placeholder="My App">
        </div>
        <div class="form-group">
          <label>Client Type</label>
          <select v-model="newClient.client_type">
            <option value="public">Public (SPA / Mobile)</option>
            <option value="confidential">Confidential (Server)</option>
          </select>
        </div>
        <div class="form-group">
          <label>Redirect URIs (one per line)</label>
          <textarea v-model="newClient.redirect_uris" rows="3" placeholder="http://localhost:3000/callback"></textarea>
        </div>
        <div class="form-group">
          <label>Grant Types</label>
          <div class="checkbox-group">
            <label class="checkbox-label">
              <input type="checkbox" value="authorization_code" v-model="newClient.grant_types"> Authorization Code
            </label>
            <label class="checkbox-label">
              <input type="checkbox" value="client_credentials" v-model="newClient.grant_types"> Client Credentials
            </label>
            <label class="checkbox-label">
              <input type="checkbox" value="refresh_token" v-model="newClient.grant_types"> Refresh Token
            </label>
          </div>
        </div>
        <div class="form-group">
          <label>Description (optional)</label>
          <input v-model="newClient.description" placeholder="What does this app do?">
        </div>
        <p v-if="clientError" class="error-msg">{{ clientError }}</p>
        <button class="btn-primary" @click="createClient">Create Client</button>

        <!-- Show secret once -->
        <div v-if="createdSecret" class="secret-display">
          <p><strong>Client Secret (shown only once):</strong></p>
          <code class="secret-code">{{ createdSecret }}</code>
          <p class="secret-warn">Copy this now. It will not be shown again.</p>
        </div>
      </div>

      <!-- Client List -->
      <p v-if="!myClients.length && !showCreateClient" class="empty">No clients created yet</p>
      <div v-for="client in myClients" :key="client.id" class="client-item-block">
        <!-- Edit mode -->
        <div v-if="editingClientId === client.id" class="edit-form">
          <div class="form-group">
            <label>Application Name</label>
            <input v-model="editClient.name">
          </div>
          <div class="form-group">
            <label>Redirect URIs (one per line)</label>
            <textarea v-model="editClient.redirect_uris" rows="3"></textarea>
          </div>
          <div class="form-group">
            <label>Grant Types</label>
            <div class="checkbox-group">
              <label class="checkbox-label">
                <input type="checkbox" value="authorization_code" v-model="editClient.grant_types"> Authorization Code
              </label>
              <label class="checkbox-label">
                <input type="checkbox" value="client_credentials" v-model="editClient.grant_types"> Client Credentials
              </label>
              <label class="checkbox-label">
                <input type="checkbox" value="refresh_token" v-model="editClient.grant_types"> Refresh Token
              </label>
            </div>
          </div>
          <p v-if="editError" class="error-msg">{{ editError }}</p>
          <div class="edit-actions">
            <button class="btn-primary btn-sm" @click="saveEditClient(client.id)">Save</button>
            <button class="btn-secondary btn-sm" @click="cancelEdit">Cancel</button>
          </div>
        </div>
        <!-- View mode -->
        <div v-else class="client-item">
          <div class="client-info">
            <div class="client-name">{{ client.name }}</div>
            <div class="client-meta">
              <code class="client-id">{{ client.client_id }}</code>
              <span class="type-badge" :class="'type-' + client.client_type">{{ client.client_type }}</span>
            </div>
            <div class="client-grants">
              <span v-for="gt in client.allowed_grant_types" :key="gt" class="grant-badge">{{ gt }}</span>
            </div>
            <div class="client-uris">
              <span v-for="uri in client.redirect_uris" :key="uri" class="uri-tag">{{ uri }}</span>
            </div>
          </div>
          <div class="client-actions">
            <button class="btn-secondary btn-sm" @click="startEditClient(client)">Edit</button>
            <button class="btn-danger btn-sm" @click="deleteClient(client.id)">Delete</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Sessions Tab -->
    <div v-if="tab === 'sessions'" class="card">
      <h2>Active Sessions</h2>
      <p v-if="!sessions.length" class="empty">No active sessions</p>
      <div v-for="s in sessions" :key="s.session_id" class="session-item">
        <div class="session-info">
          <div class="session-browser">
            {{ parseUA(s.user_agent) }}
            <span v-if="s.is_current" class="current-badge">Current</span>
          </div>
          <div class="session-meta">
            <span>IP: {{ s.ip_address || 'Unknown' }}</span>
            <span>Login: {{ new Date(s.created_at * 1000).toLocaleString('ja-JP') }}</span>
            <span>Last: {{ new Date(s.last_accessed_at * 1000).toLocaleString('ja-JP') }}</span>
          </div>
        </div>
        <button v-if="!s.is_current" class="btn-danger btn-sm" @click="revokeSession(s.session_id)">Revoke</button>
        <span v-else class="current-label">使用中</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard h1 { margin-bottom: 0.3rem; }
.welcome { color: #666; margin-bottom: 1.5rem; display: flex; align-items: center; gap: 0.5rem; }
.role-badge {
  font-size: 0.7rem; padding: 0.15rem 0.5rem; border-radius: 3px; font-weight: 600; text-transform: uppercase;
}
.role-admin { background: #fce4ec; color: #c62828; }
.role-moderator { background: #fff3e0; color: #e65100; }
.role-member { background: #e8f5e9; color: #2e7d32; }

.tabs { display: flex; gap: 0; margin-bottom: 1rem; }
.tab {
  padding: 0.6rem 1.2rem; background: #e8e8e8; border: none; cursor: pointer;
  font-size: 0.9rem; font-weight: 500; color: #555;
}
.tab:first-child { border-radius: 6px 0 0 6px; }
.tab:last-child { border-radius: 0 6px 6px 0; }
.tab.active { background: #1976d2; color: #fff; }

.profile-layout { display: grid; grid-template-columns: 1fr 200px; gap: 2rem; align-items: start; }
@media (max-width: 600px) { .profile-layout { grid-template-columns: 1fr; } }

.color-input { display: flex; align-items: center; gap: 0.8rem; }
.color-input input[type="color"] { width: 48px; height: 36px; border: 1px solid #ddd; border-radius: 4px; padding: 2px; cursor: pointer; }
.color-hex { font-family: monospace; font-size: 0.9rem; color: #555; }

.form-actions { display: flex; align-items: center; gap: 1rem; }
.save-msg { font-size: 0.85rem; color: #2e7d32; font-weight: 500; }

.preview-card {
  border: 2px solid; border-radius: 8px; padding: 1.5rem;
  text-align: center; background: #fff;
}
.preview-avatar { margin-bottom: 0.8rem; }
.preview-avatar img { width: 64px; height: 64px; border-radius: 50%; object-fit: cover; }
.avatar-placeholder {
  width: 64px; height: 64px; border-radius: 50%; margin: 0 auto;
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 1.5rem; font-weight: 700;
}
.preview-name { font-weight: 600; font-size: 1rem; }
.preview-tagline { font-size: 0.8rem; color: #888; margin-top: 0.2rem; }
.hint { font-weight: 400; color: #aaa; font-size: 0.8rem; }

h2 { margin-bottom: 1rem; }
.empty { color: #888; font-size: 0.9rem; }

.app-item, .client-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 0.8rem 0; border-bottom: 1px solid #f0f0f0;
}
.app-item:last-child, .client-item:last-child { border-bottom: none; }
.app-name, .client-name { font-weight: 600; margin-bottom: 0.2rem; }
.app-scopes { display: flex; gap: 0.3rem; flex-wrap: wrap; margin-bottom: 0.2rem; }
.scope-badge {
  display: inline-block; padding: 0.1rem 0.5rem; border-radius: 3px;
  font-size: 0.75rem; background: #e3f2fd; color: #1565c0;
}
.app-date { font-size: 0.8rem; color: #888; }

.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
.section-header h2 { margin-bottom: 0; }

.create-form {
  background: #f8f9fa; border-radius: 6px; padding: 1.2rem;
  margin-bottom: 1.5rem; border: 1px solid #e0e0e0;
}
.create-form .form-group { margin-bottom: 0.8rem; }
textarea {
  padding: 0.6rem 0.8rem; border: 1px solid #ddd; border-radius: 6px;
  font-size: 0.9rem; width: 100%; resize: vertical; font-family: inherit;
}
select {
  padding: 0.6rem 0.8rem; border: 1px solid #ddd; border-radius: 6px;
  font-size: 0.9rem; width: 100%; background: #fff;
}

.secret-display {
  margin-top: 1rem; padding: 1rem; background: #fff3cd;
  border: 1px solid #ffc107; border-radius: 6px;
}
.secret-code {
  display: block; padding: 0.6rem; background: #fff; border: 1px solid #ddd;
  border-radius: 4px; word-break: break-all; font-size: 0.85rem; margin: 0.5rem 0;
}
.secret-warn { font-size: 0.8rem; color: #856404; font-weight: 500; }

.client-meta { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.3rem; }
.client-id { font-size: 0.8rem; }
.type-badge {
  font-size: 0.7rem; padding: 0.1rem 0.4rem; border-radius: 3px; font-weight: 600;
}
.type-public { background: #e8f5e9; color: #2e7d32; }
.type-confidential { background: #fff3e0; color: #e65100; }
.client-uris { display: flex; gap: 0.3rem; flex-wrap: wrap; }
.uri-tag {
  font-size: 0.75rem; padding: 0.1rem 0.4rem; border-radius: 3px;
  background: #f0f0f0; color: #555; font-family: monospace;
}
.btn-sm { padding: 0.35rem 0.8rem; font-size: 0.82rem; }
.btn-secondary { background: #e0e0e0; color: #333; border: none; border-radius: 6px; cursor: pointer; }
.btn-secondary:hover { background: #d0d0d0; }

.checkbox-group { display: flex; flex-direction: column; gap: 0.4rem; }
.checkbox-label { display: flex; align-items: center; gap: 0.4rem; font-size: 0.9rem; cursor: pointer; }
.checkbox-label input[type="checkbox"] { cursor: pointer; }

.client-item-block { border-bottom: 1px solid #f0f0f0; }
.client-item-block:last-child { border-bottom: none; }
.client-grants { display: flex; gap: 0.3rem; flex-wrap: wrap; margin-bottom: 0.2rem; }
.grant-badge {
  display: inline-block; padding: 0.1rem 0.5rem; border-radius: 3px;
  font-size: 0.7rem; background: #f3e5f5; color: #7b1fa2; font-family: monospace;
}
.client-actions { display: flex; gap: 0.4rem; flex-shrink: 0; }

.edit-form {
  background: #f8f9fa; border-radius: 6px; padding: 1rem;
  margin: 0.5rem 0; border: 1px solid #e0e0e0;
}
.edit-form .form-group { margin-bottom: 0.8rem; }
.edit-actions { display: flex; gap: 0.5rem; }

.session-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 0.8rem 0; border-bottom: 1px solid #f0f0f0;
}
.session-item:last-child { border-bottom: none; }
.session-browser { font-weight: 600; margin-bottom: 0.2rem; }
.session-meta { display: flex; gap: 1rem; font-size: 0.8rem; color: #888; flex-wrap: wrap; }
.current-badge {
  display: inline-block; font-size: 0.7rem; padding: 0.1rem 0.4rem; border-radius: 3px;
  background: #e8f5e9; color: #2e7d32; font-weight: 600; margin-left: 0.4rem;
}
.current-label { font-size: 0.8rem; color: #888; flex-shrink: 0; }
</style>
