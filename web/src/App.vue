<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
onMounted(() => auth.fetchMe())
</script>

<template>
  <div id="app-root">
    <header class="app-header" v-if="auth.loaded">
      <div class="header-inner">
        <router-link to="/" class="logo">jyogi-oauth</router-link>
        <nav class="header-nav" v-if="auth.isLoggedIn()">
          <router-link to="/dashboard">Dashboard</router-link>
          <router-link v-if="auth.member?.role === 'admin' || auth.member?.role === 'moderator'" to="/admin">Admin</router-link>
          <button class="btn-outline btn-sm" @click="auth.logout().then(() => $router.push('/'))">Logout</button>
        </nav>
        <nav class="header-nav" v-else>
          <router-link to="/login">Login</router-link>
        </nav>
      </div>
    </header>
    <main>
      <router-view />
    </main>
  </div>
</template>

<style>
@import './style.css';

.app-header {
  background: #fff;
  border-bottom: 1px solid #e0e0e0;
  padding: 0 2rem;
}
.header-inner {
  max-width: 800px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
}
.logo {
  font-size: 1.2rem;
  font-weight: 700;
  color: #1a1a1a !important;
  text-decoration: none !important;
}
.header-nav {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.header-nav a {
  font-size: 0.9rem;
  font-weight: 500;
}
.btn-sm {
  padding: 0.35rem 0.8rem;
  font-size: 0.82rem;
}
</style>
