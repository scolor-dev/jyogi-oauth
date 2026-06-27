import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const member = ref<any>(null)
  const identity = ref<any>(null)
  const loaded = ref(false)

  async function fetchMe() {
    try {
      const data = await api.getMe()
      member.value = data.member
      identity.value = data.identity
      loaded.value = true
    } catch {
      member.value = null
      identity.value = null
      loaded.value = true
    }
  }

  async function logout() {
    await api.logout()
    member.value = null
    identity.value = null
  }

  const isLoggedIn = () => member.value !== null
  const mustChangePassword = () => member.value?.must_change_password === true

  return { member, identity, loaded, fetchMe, logout, isLoggedIn, mustChangePassword }
})
