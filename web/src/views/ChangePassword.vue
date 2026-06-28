<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import { getErrorMessage } from '../types'

const router = useRouter()
const auth = useAuthStore()

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

onMounted(async () => {
  if (!auth.loaded) await auth.fetchMe()
})

async function handleSubmit() {
  error.value = ''
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'New passwords do not match'
    return
  }
  loading.value = true
  try {
    await api.changePassword(currentPassword.value, newPassword.value)
    await auth.fetchMe()
    router.push('/dashboard')
  } catch (e: unknown) {
    error.value = getErrorMessage(e, 'Failed to change password')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="container change-pw-page">
    <div class="card change-pw-card">
      <h2>パスワード変更</h2>
      <p class="notice">パスワードの変更が必要です。新しいパスワードを設定してください。</p>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="currentPw">現在のパスワード（一時パスワード）</label>
          <input id="currentPw" v-model="currentPassword" type="password" required autofocus>
        </div>
        <div class="form-group">
          <label for="newPw">新しいパスワード</label>
          <input id="newPw" v-model="newPassword" type="password" required>
        </div>
        <div class="form-group">
          <label for="confirmPw">新しいパスワード（確認）</label>
          <input id="confirmPw" v-model="confirmPassword" type="password" required>
        </div>
        <p v-if="error" class="error-msg">{{ error }}</p>
        <button type="submit" class="btn-primary submit-btn" :disabled="loading">
          {{ loading ? '変更中...' : 'パスワードを変更' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.change-pw-page { display: flex; justify-content: center; padding-top: 4rem; }
.change-pw-card { max-width: 420px; width: 100%; }
.change-pw-card h2 { margin-bottom: 0.5rem; }
.notice { color: #e65100; font-size: 0.9rem; margin-bottom: 1.5rem; background: #fff3e0; padding: 0.6rem 0.8rem; border-radius: 4px; }
.submit-btn { width: 100%; margin-top: 0.5rem; }
</style>
