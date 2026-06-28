import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { Identity, Member } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const member = ref<Member | null>(null)
  const identity = ref<Identity | null>(null)
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
