<script setup lang="ts">
import { useAuthStore } from '../../stores/auth'
const auth = useAuthStore()
</script>

<template>
  <div>
    <h1>Admin Dashboard</h1>
    <p>ログイン中: <strong>{{ auth.member?.username }}</strong>
      <span class="role-badge" :class="'role-' + auth.member?.role">{{ auth.member?.role }}</span>
    </p>

    <div class="overview-cards">
      <router-link to="/admin/members" class="overview-card">
        <div class="card-icon">👥</div>
        <div class="card-label">Members</div>
        <div class="card-desc">メンバーの管理・ロール変更</div>
      </router-link>
      <router-link v-if="auth.member?.role === 'admin'" to="/admin/clients" class="overview-card">
        <div class="card-icon">🔑</div>
        <div class="card-label">Clients</div>
        <div class="card-desc">全OAuthクライアントの管理</div>
      </router-link>
    </div>
  </div>
</template>

<style scoped>
h1 { margin-bottom: 0.5rem; }
p { color: #666; margin-bottom: 2rem; }
.role-badge {
  font-size: 0.7rem; padding: 0.15rem 0.5rem; border-radius: 3px;
  font-weight: 600; text-transform: uppercase; margin-left: 0.5rem;
}
.role-admin { background: #fce4ec; color: #c62828; }
.role-moderator { background: #fff3e0; color: #e65100; }
.overview-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 1rem; }
.overview-card {
  background: #fff; border: 1px solid #e0e0e0; border-radius: 8px;
  padding: 1.5rem; text-decoration: none; color: inherit;
  transition: box-shadow 0.2s, border-color 0.2s;
}
.overview-card:hover { box-shadow: 0 4px 12px rgba(0,0,0,0.08); border-color: #1976d2; }
.card-icon { font-size: 2rem; margin-bottom: 0.5rem; }
.card-label { font-size: 1.1rem; font-weight: 600; margin-bottom: 0.3rem; }
.card-desc { font-size: 0.85rem; color: #888; }
</style>
