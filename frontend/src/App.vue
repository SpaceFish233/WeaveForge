<script lang="ts" setup>
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

function isActive(path: string) {
  return route.path === path
}
</script>

<template>
  <div class="app-shell">
    <nav class="nav-bar">
      <div class="nav-brand">WeaveForge</div>
      <div class="nav-links">
        <button
          class="nav-link"
          :class="{ active: isActive('/') }"
          @click="router.push('/')"
        >
          ✍ 写作
        </button>
        <button
          class="nav-link"
          :class="{ active: isActive('/settings') }"
          @click="router.push('/settings')"
        >
          ⚙ 设定管理
        </button>
        <button
          class="nav-link"
          :class="{ active: isActive('/api') }"
          @click="router.push('/api')"
        >
          🔑 API 设置
        </button>
        <button
          class="nav-link"
          :class="{ active: isActive('/foreshadow') }"
          @click="router.push('/foreshadow')"
        >
          🔮 伏笔管理
        </button>
        <button
          class="nav-link"
          :class="{ active: isActive('/plot') }"
          @click="router.push('/plot')"
        >
          🧭 剧情推演
        </button>
        <button
          class="nav-link"
          :class="{ active: isActive('/characters') }"
          @click="router.push('/characters')"
        >
          👥 角色管理
        </button>
      </div>
    </nav>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.nav-bar {
  display: flex;
  align-items: center;
  height: 40px;
  background: #161b22;
  border-bottom: 1px solid #21262d;
  padding: 0 16px;
  gap: 24px;
  flex-shrink: 0;
}

.nav-brand {
  font-size: 14px;
  font-weight: 700;
  color: #58a6ff;
  letter-spacing: 0.5px;
}

.nav-links {
  display: flex;
  gap: 4px;
}

.nav-link {
  padding: 6px 12px;
  border: none;
  background: transparent;
  color: #8b949e;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  border-radius: 6px;
  transition: all 0.15s;
  font-family: inherit;
}

.nav-link:hover {
  background: #1c2128;
  color: #c9d1d9;
}

.nav-link.active {
  background: #1f2937;
  color: #c9d1d9;
}

.app-main {
  flex: 1;
  overflow: hidden;
}
</style>
