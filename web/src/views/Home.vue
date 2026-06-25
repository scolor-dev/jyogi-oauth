<script setup lang="ts">
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
</script>

<template>
  <div class="container home">
    <div class="hero">
      <h1>jyogi-oauth</h1>
      <p class="subtitle">部内 OAuth 2.0 認可基盤</p>
      <p class="desc">メンバー認証・認可を一元管理し、部内システム間の安全な連携を実現します。</p>
      <div class="hero-actions" v-if="auth.loaded">
        <router-link v-if="auth.isLoggedIn()" to="/dashboard" class="btn-primary hero-btn">Dashboard</router-link>
        <router-link v-else to="/login" class="btn-primary hero-btn">Login</router-link>
      </div>
    </div>

    <div class="features">
      <div class="card feature">
        <h3>OAuth 2.0</h3>
        <p>Authorization Code + PKCE フローによる安全な認証・認可</p>
      </div>
      <div class="card feature">
        <h3>Identity</h3>
        <p>メンバーの表示名・アイコン・テーマカラーを管理・配信</p>
      </div>
      <div class="card feature">
        <h3>Extensible</h3>
        <p>将来のサーバー分離に対応した拡張可能なアーキテクチャ</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home { text-align: center; padding-top: 4rem; }
.hero { margin-bottom: 3rem; }
.hero h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
.subtitle { font-size: 1.2rem; color: #666; margin-bottom: 0.8rem; }
.desc { color: #888; max-width: 500px; margin: 0 auto 1.5rem; }
.hero-btn {
  display: inline-block; padding: 0.8rem 2rem; border-radius: 8px;
  font-size: 1rem; font-weight: 600; color: #fff; background: #1976d2;
}
.hero-btn:hover { text-decoration: none; opacity: 0.9; }
.features { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-top: 2rem; }
.feature { text-align: left; }
.feature h3 { margin-bottom: 0.4rem; color: #1976d2; }
.feature p { font-size: 0.9rem; color: #666; }
</style>
