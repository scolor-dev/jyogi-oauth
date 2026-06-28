<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import { getErrorMessage } from '../types'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    const res = await api.login(username.value, password.value)

    if (res.redirect_to && res.redirect_to !== '/') {
      if (res.redirect_to.startsWith('http') || res.redirect_to.startsWith('/oauth/')) {
        window.location.href = res.redirect_to
      } else {
        router.push(res.redirect_to)
      }
      return
    }

    await auth.fetchMe()
    router.push('/dashboard')
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Login failed')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="container login-page">
    <div class="card login-card">
      <h2>Login</h2>
      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label for="username">Username</label>
          <input id="username" v-model="username" type="text" autocomplete="username" required autofocus>
        </div>
        <div class="form-group">
          <label for="password">Password</label>
          <input id="password" v-model="password" type="password" autocomplete="current-password" required>
        </div>
        <p v-if="error" class="error-msg">{{ error }}</p>
        <button type="submit" class="btn-primary login-btn" :disabled="loading">
          {{ loading ? 'Logging in...' : 'Login' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page { display: flex; justify-content: center; padding-top: 4rem; }
.login-card { max-width: 400px; width: 100%; }
.login-card h2 { margin-bottom: 1.5rem; }
.login-btn { width: 100%; margin-top: 0.5rem; }
</style>
