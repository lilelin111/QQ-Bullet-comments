<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  EventsOff,
  EventsOn,
  Quit,
  WindowFullscreen,
  WindowIsFullscreen,
  WindowUnfullscreen,
} from '../wailsjs/runtime/runtime'
import * as api from './api.js'

const STORAGE_KEY = 'qq-danmaku-appearance'

const defaults = {
  background: '#0f172a',
  title: '#22d3ee',
  content: '#f8fafc',
  scale: 1,
}

function loadAppearance() {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
    return {
      background: validColor(saved.background, defaults.background),
      title: validColor(saved.title, defaults.title),
      content: validColor(saved.content, defaults.content),
      scale: boundedScale(saved.scale),
    }
  } catch {
    return { ...defaults }
  }
}

function validColor(value, fallback) {
  return /^#[0-9a-f]{6}$/i.test(String(value || '')) ? value : fallback
}

function boundedScale(value) {
  const number = Number(value)
  return Number.isFinite(number) ? Math.min(2, Math.max(0.75, number)) : defaults.scale
}

const appearance = reactive(loadAppearance())
const settingsOpen = ref(true)
const isFullscreen = ref(true)
const currentUser = ref(null)
const authMode = ref('login')
const authBusy = ref(false)
const authError = ref('')
const authForm = reactive({
  username: '',
  password: '',
})
const bullets = ref([])
const notice = ref('')
const laneStep = ref(88)
const pendingMessages = []
const laneReadyAt = []

let nextBulletId = 1
let queueTimer = null
let noticeTimer = null

const themeStyle = computed(() => ({
  '--bullet-background': appearance.background,
  '--bullet-title': appearance.title,
  '--bullet-content': appearance.content,
  '--bullet-scale': String(appearance.scale),
}))

watch(
  appearance,
  (value) => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
    updateLanes()
  },
  { deep: true },
)

function unwrapPayload(payload) {
  let current = payload
  while (Array.isArray(current) && current.length === 1) {
    current = current[0]
  }
  if (Array.isArray(current)) {
    return current[0] ?? {}
  }
  return current ?? {}
}

function normalizeMessage(payload) {
  const value = unwrapPayload(payload)

  if (typeof value === 'string') {
    // 后端原始通知格式为：ID<TAB>时间<TAB>群名<TAB>内容。
    const fields = value.split('\t', 4)
    if (fields.length === 4) {
      return {
        title: fields[2].trim(),
        content: fields[3].trim(),
      }
    }
    return { title: 'QQ消息', content: value.trim() }
  }

  return {
    title: String(value.title ?? value.group_name ?? value.groupName ?? 'QQ消息').trim(),
    content: String(value.body ?? value.message ?? value.content ?? '').trim(),
  }
}

function pushNotice(message) {
  notice.value = message
  window.clearTimeout(noticeTimer)
  noticeTimer = window.setTimeout(() => {
    notice.value = ''
  }, 4200)
}

function handleQQMessage(payload) {
  const message = normalizeMessage(payload)
  if (!message.content && !message.title) {
    return
  }

  pendingMessages.push(message)
  pumpQueue()
}

function handleMonitorError(payload) {
  const message = unwrapPayload(payload)
  pushNotice(String(message || 'QQ 通知监听失败'))
}

function updateLanes() {
  const availableHeight = Math.max(240, window.innerHeight - 20)
  const nextStep = Math.round(88 * appearance.scale)
  const nextCount = Math.max(3, Math.floor(availableHeight / nextStep))

  laneStep.value = nextStep

  while (laneReadyAt.length < nextCount) {
    laneReadyAt.push(0)
  }
  if (laneReadyAt.length > nextCount) {
    laneReadyAt.length = nextCount
  }

  pumpQueue()
}

function pumpQueue() {
  window.clearTimeout(queueTimer)
  queueTimer = null

  const now = performance.now()
  let lane = laneReadyAt.findIndex((readyAt) => readyAt <= now)

  while (pendingMessages.length > 0 && lane >= 0) {
    const message = pendingMessages.shift()
    const duration = bulletDuration(message)
    const id = nextBulletId++

    bullets.value.push({
      id,
      title: message.title || 'QQ消息',
      content: message.content || message.title || '新消息',
      top: 10 + lane * laneStep.value,
      duration,
    })

    // 留出短暂间隔，避免同一轨道连续弹幕紧贴在一起。
    laneReadyAt[lane] = now + duration * 1000 + 320
    lane = laneReadyAt.findIndex((readyAt) => readyAt <= now)
  }

  if (pendingMessages.length > 0 && laneReadyAt.length > 0) {
    const nextReadyAt = Math.min(...laneReadyAt)
    queueTimer = window.setTimeout(pumpQueue, Math.max(80, nextReadyAt - performance.now()))
  }
}

function bulletDuration(message) {
  const textLength = `${message.title}${message.content}`.length
  const estimatedWidth = Math.min(1440, 120 + textLength * 18)
  const travelDistance = window.innerWidth + estimatedWidth
  const speed = 210 + Math.min(70, textLength * 0.8)
  return Math.min(16, Math.max(6.5, travelDistance / speed))
}

function removeBullet(id) {
  bullets.value = bullets.value.filter((bullet) => bullet.id !== id)
}

function clearBullets() {
  pendingMessages.length = 0
  bullets.value = []
  laneReadyAt.fill(0)
}

function resetAppearance() {
  Object.assign(appearance, defaults)
}

function playPreview() {
  handleQQMessage({
    title: '测试群',
    body: '这是一条桌面弹幕漂浮效果预览',
  })
}

function switchAuthMode(mode) {
  authMode.value = mode
  authError.value = ''
  authForm.password = ''
}

async function submitAuth() {
  const username = authForm.username.trim()
  if (!username || !authForm.password) {
    authError.value = '请输入用户名和密码'
    return
  }

  authBusy.value = true
  authError.value = ''
  try {
    const caller = authMode.value === 'login' ? api.login : api.register
    const result = await caller(username, authForm.password)
    if (!result.success || !result.user) {
      authError.value = result.message
      return
    }

    currentUser.value = result.user
    authForm.password = ''
    pushNotice(result.message)
  } catch (error) {
    authError.value = error?.message || '账号操作失败'
  } finally {
    authBusy.value = false
  }
}

async function logoutUser() {
  authBusy.value = true
  authError.value = ''
  try {
    const result = await api.logout()
    if (!result.success) {
      authError.value = result.message
      return
    }

    currentUser.value = null
    authMode.value = 'login'
    authForm.password = ''
    clearBullets()
    pushNotice(result.message)
  } catch (error) {
    authError.value = error?.message || '退出登录失败'
  } finally {
    authBusy.value = false
  }
}

async function toggleWindowMode() {
  try {
    if (isFullscreen.value) {
      WindowUnfullscreen()
    } else {
      WindowFullscreen()
    }

    await new Promise((resolve) => window.setTimeout(resolve, 180))
    isFullscreen.value = await WindowIsFullscreen()
    updateLanes()
  } catch {
    pushNotice('无法切换窗口模式')
  }
}

function exitApp() {
  try {
    Quit()
  } catch {
    pushNotice('当前环境无法退出应用')
  }
}

onMounted(async () => {
  try {
    isFullscreen.value = await WindowIsFullscreen()
  } catch {
    isFullscreen.value = true
  }
  updateLanes()
  window.addEventListener('resize', updateLanes)
  EventsOn('qq:new-message', handleQQMessage)
  EventsOn('qq:monitor-error', handleMonitorError)
})

onBeforeUnmount(() => {
  window.clearTimeout(queueTimer)
  window.clearTimeout(noticeTimer)
  window.removeEventListener('resize', updateLanes)
  EventsOff('qq:new-message')
  EventsOff('qq:monitor-error')
})
</script>

<template>
  <main class="overlay" :style="themeStyle">
    <div
      v-if="!isFullscreen"
      class="window-drag-handle"
      title="拖动移动窗口"
      aria-label="拖动移动窗口"
    ></div>

    <section class="danmaku-layer" aria-live="polite">
      <article
        v-for="bullet in bullets"
        :key="bullet.id"
        class="danmaku"
        :style="{
          '--bullet-top': `${bullet.top}px`,
          '--bullet-duration': `${bullet.duration}s`,
        }"
        @animationend="removeBullet(bullet.id)"
      >
        <strong class="danmaku-title">{{ bullet.title }}</strong>
        <span class="danmaku-separator" aria-hidden="true"></span>
        <span class="danmaku-content">{{ bullet.content }}</span>
      </article>
    </section>

    <aside class="quick-settings">
      <button
        v-if="!settingsOpen"
        class="settings-trigger"
        type="button"
        @click="settingsOpen = true"
      >
        {{ currentUser ? '调节' : '登录' }}
      </button>

      <section v-else class="settings-panel">
        <header class="settings-head">
          <div class="monitor-state">
            <span class="monitor-dot" :class="{ offline: !currentUser }"></span>
            <h1>
              {{ currentUser ? '桌面弹幕' : (authMode === 'login' ? '用户登录' : '用户注册') }}
            </h1>
          </div>
          <div class="settings-head-actions">
            <button class="text-button" type="button" @click="toggleWindowMode">
              {{ isFullscreen ? '移动窗口' : '铺满屏幕' }}
            </button>
            <button class="text-button" type="button" @click="settingsOpen = false">
              收起
            </button>
          </div>
        </header>

        <template v-if="!currentUser">
          <div class="auth-switch">
            <button
              type="button"
              :class="{ active: authMode === 'login' }"
              @click="switchAuthMode('login')"
            >
              登录
            </button>
            <button
              type="button"
              :class="{ active: authMode === 'register' }"
              @click="switchAuthMode('register')"
            >
              注册
            </button>
          </div>

          <form class="auth-form" @submit.prevent="submitAuth">
            <label class="field">
              <span>用户名</span>
              <input
                v-model.trim="authForm.username"
                type="text"
                autocomplete="username"
                maxlength="32"
              />
            </label>

            <label class="field">
              <span>密码</span>
              <input
                v-model="authForm.password"
                type="password"
                :autocomplete="authMode === 'login' ? 'current-password' : 'new-password'"
                minlength="8"
                maxlength="16"
              />
            </label>

            <p v-if="authMode === 'register'" class="auth-hint">
              8~16 位，需包含大小写字母，以及数字或符号
            </p>
            <p v-if="authError" class="auth-error">{{ authError }}</p>

            <button class="auth-submit" type="submit" :disabled="authBusy">
              {{ authBusy ? '处理中…' : (authMode === 'login' ? '登录' : '注册') }}
            </button>
          </form>
        </template>

        <template v-else>
          <div class="user-strip">
            <div>
              <strong>{{ currentUser.name }}</strong>
              <span>ID {{ currentUser.id }}</span>
            </div>
            <button type="button" :disabled="authBusy" @click="logoutUser">退出登录</button>
          </div>

          <div class="settings-list">
          <label class="color-control">
            <span>弹幕背景</span>
            <span class="color-value">
              <code>{{ appearance.background }}</code>
              <input v-model="appearance.background" type="color" />
            </span>
          </label>

          <label class="color-control">
            <span>群名颜色</span>
            <span class="color-value">
              <code>{{ appearance.title }}</code>
              <input v-model="appearance.title" type="color" />
            </span>
          </label>

          <label class="color-control">
            <span>内容颜色</span>
            <span class="color-value">
              <code>{{ appearance.content }}</code>
              <input v-model="appearance.content" type="color" />
            </span>
          </label>

          <label class="size-control">
            <span>弹幕大小</span>
            <output>{{ Math.round(appearance.scale * 100) }}%</output>
            <input v-model.number="appearance.scale" type="range" min="0.75" max="2" step="0.05" />
          </label>
        </div>

          <div class="preview" aria-hidden="true">
            <strong>群名称</strong>
            <span></span>
            <p>QQ 消息内容预览</p>
          </div>

          <footer class="settings-actions">
            <button type="button" @click="resetAppearance">恢复默认</button>
            <button type="button" @click="playPreview">试放弹幕</button>
            <button type="button" @click="clearBullets">清空弹幕</button>
            <button class="quit-button" type="button" @click="exitApp">退出</button>
          </footer>
        </template>
      </section>
    </aside>

    <div v-if="notice" class="notice" role="status">{{ notice }}</div>
  </main>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  overflow: hidden;
  background: transparent;
  color: #f8fafc;
}

.window-drag-handle {
  --wails-draggable: drag;
  position: fixed;
  top: 8px;
  left: 50%;
  z-index: 40;
  width: 72px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 1px solid #334155;
  border-radius: 8px;
  background: rgba(8, 15, 28, 0.94);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.32);
  cursor: grab;
  transform: translateX(-50%);
}

.window-drag-handle::before {
  width: 30px;
  height: 10px;
  content: "";
  background-image: radial-gradient(circle, #cbd5e1 1.3px, transparent 1.5px);
  background-size: 7px 5px;
}

.window-drag-handle:active {
  cursor: grabbing;
}

.danmaku-layer {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.danmaku {
  position: absolute;
  top: var(--bullet-top);
  left: 100vw;
  display: inline-flex;
  align-items: baseline;
  gap: calc(12px * var(--bullet-scale));
  max-width: min(90vw, 1440px);
  padding: calc(9px * var(--bullet-scale)) calc(15px * var(--bullet-scale));
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--bullet-background) 62%, white);
  border-radius: 8px;
  background: var(--bullet-background);
  box-shadow: 0 8px 26px rgba(0, 0, 0, 0.3);
  backface-visibility: hidden;
  contain: layout paint;
  white-space: nowrap;
  transform: translate3d(0, 0, 0);
  animation: danmaku-move var(--bullet-duration) linear both;
  will-change: transform, opacity;
}

.danmaku-title {
  flex: 0 0 auto;
  color: var(--bullet-title);
  font-size: calc(19px * var(--bullet-scale));
  line-height: 1.25;
}

.danmaku-separator {
  width: 1px;
  height: calc(22px * var(--bullet-scale));
  flex: 0 0 auto;
  align-self: center;
  background: color-mix(in srgb, var(--bullet-title) 55%, transparent);
}

.danmaku-content {
  min-width: 0;
  overflow: hidden;
  color: var(--bullet-content);
  font-size: calc(16px * var(--bullet-scale));
  line-height: 1.35;
  text-overflow: ellipsis;
}

@keyframes danmaku-move {
  0% {
    opacity: 0;
    transform: translate3d(0, 0, 0);
  }
  3% {
    opacity: 1;
  }
  97% {
    opacity: 1;
  }
  100% {
    opacity: 0.88;
    transform: translate3d(calc(-100vw - 100% - 20px), 0, 0);
  }
}

.quick-settings {
  position: fixed;
  top: 18px;
  right: 18px;
  z-index: 10;
}

.settings-trigger,
.settings-panel {
  border: 1px solid #334155;
  border-radius: 8px;
  background: rgba(8, 15, 28, 0.94);
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.38);
  backdrop-filter: blur(14px);
}

.settings-trigger {
  height: 38px;
  padding: 0 16px;
  color: #e2e8f0;
  font-size: 14px;
  font-weight: 700;
}

.settings-panel {
  width: min(342px, calc(100vw - 36px));
  padding: 16px;
}

.settings-head,
.monitor-state,
.color-control,
.color-value,
.size-control,
.settings-actions {
  display: flex;
  align-items: center;
}

.settings-head {
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.settings-head-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.monitor-state {
  gap: 9px;
}

.monitor-state h1 {
  margin: 0;
  color: #f8fafc;
  font-size: 16px;
  line-height: 1.2;
}

.monitor-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 0 4px rgba(34, 197, 94, 0.14);
}

.monitor-dot.offline {
  background: #64748b;
  box-shadow: 0 0 0 4px rgba(100, 116, 139, 0.16);
}

.text-button {
  height: 30px;
  padding: 0 9px;
  border: 0;
  background: transparent;
  color: #94a3b8;
  font-size: 13px;
}

.auth-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  margin-bottom: 15px;
  padding: 4px;
  border: 1px solid #1e293b;
  border-radius: 8px;
  background: #0b1424;
}

.auth-switch button {
  height: 34px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #94a3b8;
  font-size: 13px;
}

.auth-switch button.active {
  background: #173042;
  color: #67e8f9;
  font-weight: 700;
}

.auth-form {
  display: grid;
  gap: 13px;
}

.field {
  display: grid;
  gap: 7px;
}

.field > span {
  color: #94a3b8;
  font-size: 12px;
}

.field input {
  width: 100%;
  height: 40px;
  padding: 0 12px;
  border: 1px solid #334155;
  border-radius: 6px;
  background: #0b1424;
  color: #f8fafc;
  outline: none;
}

.field input:focus {
  border-color: #38bdf8;
}

.auth-hint,
.auth-error {
  margin: -2px 0 0;
  font-size: 12px;
  line-height: 1.5;
}

.auth-hint {
  color: #94a3b8;
}

.auth-error {
  color: #fca5a5;
}

.auth-submit {
  height: 40px;
  border: 0;
  border-radius: 6px;
  background: #22d3ee;
  color: #07131d;
  font-size: 14px;
  font-weight: 700;
}

.auth-submit:disabled,
.user-strip button:disabled {
  cursor: wait;
  opacity: 0.58;
}

.user-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding: 10px 11px;
  border: 1px solid #1e293b;
  border-radius: 7px;
  background: #0b1424;
}

.user-strip > div {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.user-strip strong {
  overflow: hidden;
  color: #f8fafc;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-strip span {
  color: #64748b;
  font-size: 11px;
}

.user-strip button {
  height: 30px;
  flex: 0 0 auto;
  padding: 0 10px;
  border: 1px solid #475569;
  border-radius: 6px;
  background: transparent;
  color: #cbd5e1;
  font-size: 12px;
}

.settings-list {
  display: grid;
  gap: 6px;
}

.color-control,
.size-control {
  min-height: 48px;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid #1e293b;
  color: #cbd5e1;
  font-size: 13px;
}

.color-value {
  gap: 10px;
}

.color-value code {
  color: #94a3b8;
  font-family: "Cascadia Mono", Consolas, monospace;
  font-size: 11px;
  text-transform: uppercase;
}

input[type='color'] {
  width: 42px;
  height: 30px;
  padding: 2px;
  border: 1px solid #475569;
  border-radius: 6px;
  background: #0f172a;
  cursor: pointer;
}

.size-control {
  display: grid;
  grid-template-columns: 1fr auto;
  padding-top: 10px;
}

.size-control output {
  color: #67e8f9;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.size-control input {
  grid-column: 1 / -1;
  width: 100%;
  margin: 7px 0 9px;
  accent-color: #22d3ee;
}

.preview {
  display: flex;
  align-items: baseline;
  gap: 9px;
  margin-top: 15px;
  padding: calc(8px * var(--bullet-scale)) calc(12px * var(--bullet-scale));
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--bullet-background) 62%, white);
  border-radius: 8px;
  background: var(--bullet-background);
  white-space: nowrap;
}

.preview strong {
  flex: 0 0 auto;
  color: var(--bullet-title);
  font-size: calc(16px * var(--bullet-scale));
}

.preview span {
  width: 1px;
  height: calc(19px * var(--bullet-scale));
  flex: 0 0 auto;
  background: color-mix(in srgb, var(--bullet-title) 55%, transparent);
}

.preview p {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--bullet-content);
  font-size: calc(14px * var(--bullet-scale));
  text-overflow: ellipsis;
}

.settings-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 14px;
}

.settings-actions button {
  height: 34px;
  flex: 1;
  padding: 0 10px;
  border: 1px solid #334155;
  border-radius: 6px;
  background: #111c2e;
  color: #cbd5e1;
  font-size: 12px;
}

.settings-actions .quit-button {
  border-color: #7f1d1d;
  color: #fca5a5;
}

.settings-trigger:hover,
.settings-actions button:hover {
  filter: brightness(1.16);
}

.notice {
  position: fixed;
  right: 18px;
  bottom: 18px;
  z-index: 20;
  max-width: min(480px, calc(100vw - 36px));
  padding: 11px 14px;
  border: 1px solid #7f1d1d;
  border-radius: 8px;
  background: rgba(69, 10, 10, 0.96);
  color: #fecaca;
  font-size: 13px;
  box-shadow: 0 12px 36px rgba(0, 0, 0, 0.36);
}

button:focus-visible,
input:focus-visible {
  outline: 2px solid #38bdf8;
  outline-offset: 2px;
}

@media (max-width: 560px) {
  .quick-settings {
    top: 10px;
    right: 10px;
  }

  .settings-panel {
    width: calc(100vw - 20px);
  }
}
</style>
