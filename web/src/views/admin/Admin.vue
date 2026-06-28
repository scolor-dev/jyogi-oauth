<script setup lang="ts">
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
</script>

<template>
  <div class="admin-layout" v-if="auth.member?.role === 'admin' || auth.member?.role === 'moderator'">
    <aside class="admin-sidebar">
      <div class="sidebar-title">Admin</div>
      <nav>
        <router-link to="/admin" exact-active-class="active">Overview</router-link>
        <router-link to="/admin/members" active-class="active">Members</router-link>
        <router-link v-if="auth.member?.role === 'admin'" to="/admin/clients" active-class="active">Clients</router-link>
        <router-link v-if="auth.member?.role === 'admin'" to="/admin/scopes" active-class="active">Scopes</router-link>
        <router-link to="/admin/audit-logs" active-class="active">Audit Logs</router-link>
      </nav>
    </aside>
    <main class="admin-main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.admin-layout { display: flex; min-height: calc(100vh - 57px); }
.admin-sidebar {
  width: 200px; background: #1a1a2e; padding: 1.5rem 0; flex-shrink: 0;
}
.sidebar-title {
  color: #fff; font-size: 1.1rem; font-weight: 700;
  padding: 0 1.2rem; margin-bottom: 1.5rem;
}
.admin-sidebar nav { display: flex; flex-direction: column; }
.admin-sidebar a {
  color: #aaa; padding: 0.6rem 1.2rem; font-size: 0.9rem;
  text-decoration: none; transition: all 0.15s;
}
.admin-sidebar a:hover { color: #fff; background: rgba(255,255,255,0.05); }
.admin-sidebar a.active { color: #fff; background: rgba(255,255,255,0.1); border-left: 3px solid #1976d2; }
.admin-main { flex: 1; padding: 2rem; max-width: 900px; }
</style>
