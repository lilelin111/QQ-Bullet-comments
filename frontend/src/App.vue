<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { EventsOff, EventsOn } from '../wailsjs/runtime/runtime'
import * as api from './api.js'

const mode = ref('login')
const credentials = reactive({
  username: '',
  password: '',
})
const currentUser = ref(null)
const queryId = ref(0)
const queryResult = reactive({
  group: '',
  message: '',
})
const status = ref({ kind: 'idle', text: '就绪' })
const activity = ref([])
const records = ref([])
const bullets = ref([])
const nextRecordId = ref(1)
const nextBulletId = ref(1)
const bulletLane = ref(0)

const bulletColor = ref('#ffffff')
const accentColor = ref('#2dd4bf')
// 使用 CSS 变量把用户从调色盘选择的颜色应用到整个桌面端界面和弹幕文字。
const themeStyle = computed(() => ({
  '--accent': accentColor.value,
  '--accent-soft': hexToRgba(accentColor.value, 0.13),
  '--bullet-color': bulletColor.value,
}))

function hexToRgba(hex, alpha) {
  const value = hex.replace('#', '')
  const full = value.length === 3
    ? value.split('').map((char) => char + char).join('')
    : value
  const number = Number.parseInt(full, 16)
  return `rgba(${(number >> 16) & 255}, ${(number >> 8) & 255}, ${number & 255}, ${alpha})`
}

function nowText() {
  return new Date().toLocaleTimeString('zh-CN', { hour12: false })
}

function setStatus(kind, text) {
  status.value = { kind, text }
}

function pushActivity(text) {
  activity.value.unshift({
    text,
    time: nowText(),
  })
  if (activity.value.length > 9) {
    activity.value.pop()
  }
}

function switchMode(next) {
  mode.value = next
  status.value = { kind: 'idle', text: '就绪' }
}

async function submitAuth() {
  if (!credentials.username.trim() || !credentials.password) {
    setStatus('error', '用户名和密码不能为空')
    return
  }

  const caller = mode.value === 'login' ? api.login : api.register
  const result = await caller(credentials.username.trim(), credentials.password)

  if (!result.success || !result.user) {
    setStatus('error', result.message)
    return
  }

  currentUser.value = result.user
  queryId.value = 0
  queryResult.group = ''
  queryResult.message = ''
  setStatus('success', result.message)
  pushActivity(`${mode.value === 'login' ? '登录' : '注册'}：${currentUser.value.name}`)
}

function logout() {
  currentUser.value = null
  records.value = []
  setStatus('idle', '已退出账号')
}

function handleQQMessage(payload) {
  // Wails 事件数据可能是单个对象，也可能包装在参数数组中，这里统一兼容。
  const msg = Array.isArray(payload) ? (payload[0] || {}) : (payload || {})
  const group = String(msg.title ?? msg.group_name ?? 'QQ消息')
  const content = String(msg.body ?? msg.message ?? '')
  if (!group && !content) {
    return
  }

  addRecord(group, content)
  spawnBullet(group, content)
  pushActivity(`收到弹幕：${group}`)
  setStatus('success', '收到新 QQ 消息')
}

function handleMonitorError(payload) {
  // 后端监听失败时把错误显示在状态栏和动态列表中，避免静默失败。
  const message = Array.isArray(payload) ? String(payload[0] ?? '') : String(payload ?? '')
  if (!message) {
    return
  }
  setStatus('error', message)
  pushActivity(`监听异常：${message}`)
}

onMounted(() => {
  // 后端发现新的 QQ 通知后，通过该事件把群名和内容推送给弹幕层。
  EventsOn('qq:new-message', handleQQMessage)
  // 通知权限关闭或查询失败时，通过该事件把原因显示给用户。
  EventsOn('qq:monitor-error', handleMonitorError)
  setStatus('info', '正在监听 QQ 通知…')
})

onBeforeUnmount(() => {
  EventsOff('qq:new-message')
  EventsOff('qq:monitor-error')
})

function addRecord(group, content) {
  // 把实时收到的 QQ 消息追加到界面记录列表，最新消息显示在最上方。
  records.value.unshift({
    id: nextRecordId.value++,
    group,
    content,
    time: nowText(),
  })
}

function spawnBullet(group, content) {
  // 为每条消息分配一个弹道，生成从右向左漂浮的弹幕节点。
  const id = nextBulletId.value++
  const lane = bulletLane.value++ % 6
  bullets.value.push({
    id,
    group,
    content,
    top: 92 + lane * 48,
  })
  window.setTimeout(() => {
    bullets.value = bullets.value.filter((bullet) => bullet.id !== id)
  }, 10000)
}

async function fetchQuery(kind) {
  const id = Number(queryId.value)
  if (!Number.isInteger(id) || id < 0) {
    setStatus('error', '请输入有效的消息编号')
    return
  }

  const caller = kind === 'group' ? api.showGetTitle : api.showGetMessage
  const result = await caller(currentUser.value.id, id)

  if (!result.success) {
    setStatus('error', result.message)
    return
  }

  if (kind === 'group') {
    queryResult.group = result.value
  } else {
    queryResult.message = result.value
  }
  setStatus('success', `已读取第 ${id} 条消息`)
}
</script>

<template>
  <div class="app" :style="themeStyle">
    <header class="topbar">
      <div class="brand">
        <div class="brand-mark">Q弹</div>
        <div class="brand-text">
          <h1>QQ 弹幕</h1>
          <p>通知捕获台</p>
        </div>
      </div>

      <div v-if="currentUser" class="session">
        <div class="session-user">
          <strong>{{ currentUser.name }}</strong>
          <span>ID {{ currentUser.id }}</span>
        </div>
        <button class="ghost-btn" type="button" @click="logout">退出</button>
      </div>
    </header>

    <!-- 弹幕浮层不拦截鼠标，消息会在这里从右侧漂浮到左侧。 -->
    <div class="bullet-layer" aria-hidden="true">
      <span
        v-for="bullet in bullets"
        :key="bullet.id"
        class="bullet"
        :style="{ top: bullet.top + 'px' }"
      >
        <em class="bullet-group">{{ bullet.group }}</em>
        <span class="bullet-content">{{ bullet.content || bullet.group }}</span>
      </span>
    </div>

    <main class="page">
      <section v-if="!currentUser" class="auth-wrap">
        <div class="auth-panel">
          <div class="segmented" role="tablist" aria-label="账号操作">
            <button
              type="button"
              role="tab"
              :aria-selected="mode === 'login'"
              :class="{ active: mode === 'login' }"
              @click="switchMode('login')"
            >
              登录
            </button>
            <button
              type="button"
              role="tab"
              :aria-selected="mode === 'register'"
              :class="{ active: mode === 'register' }"
              @click="switchMode('register')"
            >
              注册
            </button>
          </div>

          <form class="form" @submit.prevent="submitAuth">
            <label class="field">
              <span>用户名</span>
              <input
                v-model="credentials.username"
                type="text"
                autocomplete="username"
                maxlength="32"
              />
            </label>
            <label class="field">
              <span>密码</span>
              <input
                v-model="credentials.password"
                type="password"
                :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
                minlength="8"
                maxlength="16"
              />
            </label>
            <button class="primary-btn full" type="submit">
              {{ mode === 'login' ? '登录' : '注册' }}
            </button>
          </form>
        </div>
      </section>

      <section v-else class="workspace">
        <div class="side-column">
          <section class="panel">
            <div class="panel-head">
              <h2>实时监听</h2>
              <span class="live-dot waiting"></span>
            </div>
            <p class="capture-state">QQ 通知监听中</p>
          </section>

          <section class="panel">
            <div class="panel-head">
              <h2>外观设置</h2>
            </div>
            <label class="color-field">
              <span>弹幕文字颜色</span>
              <input v-model="bulletColor" type="color" />
            </label>
            <label class="color-field">
              <span>界面主题颜色</span>
              <input v-model="accentColor" type="color" />
            </label>
          </section>

          <section class="panel">
            <div class="panel-head">
              <h2>消息查询</h2>
            </div>
            <label class="field">
              <span>消息编号</span>
              <input v-model.number="queryId" type="number" min="0" step="1" />
            </label>
            <div class="btn-row">
              <button class="secondary-btn" type="button" @click="fetchQuery('group')">
                读取群名
              </button>
              <button class="secondary-btn" type="button" @click="fetchQuery('message')">
                读取内容
              </button>
            </div>
            <div class="query-result">
              <div>
                <span class="label-text">群名</span>
                <p>{{ queryResult.group || '—' }}</p>
              </div>
              <div>
                <span class="label-text">内容</span>
                <p>{{ queryResult.message || '—' }}</p>
              </div>
            </div>
          </section>
        </div>

        <section class="records-column">
          <div class="panel-head records-head">
            <h2>最近记录</h2>
            <span class="count">{{ records.length }}</span>
          </div>

          <div class="records-list">
            <article v-for="(record, index) in records" :key="record.id" class="record">
              <div class="record-index">{{ index + 1 }}</div>
              <div class="record-main">
                <strong>{{ record.group || 'QQ消息' }}</strong>
                <span class="record-content">{{ record.content || record.group }}</span>
                <span class="record-time">{{ record.time }}</span>
              </div>
            </article>
            <div v-if="records.length === 0" class="empty">
              <p>暂无记录</p>
            </div>
          </div>

          <div class="activity-panel">
            <div class="panel-head">
              <h2>动态</h2>
            </div>
            <ol class="activity-list">
              <li v-for="(item, index) in activity" :key="item.time + index">
                <time>{{ item.time }}</time>
                <span>{{ item.text }}</span>
              </li>
              <li v-if="activity.length === 0" class="empty-li">暂无动态</li>
            </ol>
          </div>
        </section>
      </section>

      <div class="status-bar" :class="status.kind">
        <span class="status-dot"></span>
        <span>{{ status.text }}</span>
      </div>
    </main>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.topbar {
  height: 68px;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 20px;
  border-bottom: 1px solid var(--line);
  background: var(--panel);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.brand-mark {
  width: 44px;
  height: 44px;
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: var(--accent);
  color: #08120f;
  font-size: 14px;
  font-weight: 800;
  letter-spacing: 0;
}

.brand-text {
  min-width: 0;
}

.brand h1 {
  margin: 0;
  font-size: 18px;
  line-height: 1.2;
}

.brand p {
  margin: 3px 0 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.2;
}

.session {
  display: flex;
  align-items: center;
  gap: 14px;
}

.session-user {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  min-width: 0;
}

.session-user strong {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.session-user span {
  color: var(--muted);
  font-size: 12px;
}

.page {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 18px;
}

.auth-wrap {
  flex: 1;
  display: grid;
  place-items: center;
  padding-bottom: 40px;
}

.auth-panel {
  width: min(420px, 100%);
  padding: 18px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
}

.segmented {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  padding: 4px;
  margin-bottom: 18px;
  border-radius: 8px;
  background: var(--inner);
}

.segmented button {
  height: 36px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 14px;
}

.segmented button.active {
  background: var(--accent-soft);
  color: var(--accent);
  font-weight: 700;
}

.form {
  display: grid;
  gap: 14px;
}

.field {
  display: grid;
  gap: 7px;
  min-width: 0;
}

.field span,
.label-text {
  color: var(--muted);
  font-size: 12px;
}

.field input {
  width: 100%;
  height: 40px;
  min-width: 0;
  padding: 0 12px;
  border: 1px solid var(--line-strong);
  border-radius: 6px;
  background: var(--input);
  color: var(--text);
  outline: none;
}

.field input:focus {
  border-color: var(--accent);
}

.workspace {
  flex: 1;
  display: grid;
  grid-template-columns: minmax(300px, 360px) minmax(0, 1fr);
  gap: 16px;
  min-height: 0;
}

.side-column {
  display: grid;
  align-content: start;
  gap: 16px;
  min-width: 0;
}

.panel {
  padding: 16px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
  min-width: 0;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.panel-head h2 {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
}

.primary-btn,
.secondary-btn,
.ghost-btn {
  height: 38px;
  padding: 0 14px;
  border-radius: 6px;
  font-size: 14px;
}

.primary-btn {
  border: 0;
  background: var(--accent);
  color: #08120f;
  font-weight: 700;
}

.primary-btn:disabled {
  cursor: wait;
  opacity: 0.62;
}

.secondary-btn {
  border: 1px solid var(--line-strong);
  background: var(--inner);
  color: var(--text);
}

.ghost-btn {
  border: 1px solid var(--line-strong);
  background: transparent;
  color: var(--muted);
}

.primary-btn:hover,
.secondary-btn:hover,
.ghost-btn:hover {
  filter: brightness(1.12);
}

.full {
  width: 100%;
}

.btn-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-bottom: 14px;
}

.live-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--danger);
}

.live-dot.waiting {
  background: var(--accent);
  animation: pulse 1.1s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 0.35;
  }
  50% {
    opacity: 1;
  }
}

.query-result {
  display: grid;
  gap: 10px;
}

.query-result div {
  padding: 10px 12px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--inner);
}

.query-result p {
  margin: 6px 0 0;
  overflow-wrap: anywhere;
  font-size: 14px;
  line-height: 1.5;
}

.records-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  min-height: 0;
}

.records-head {
  margin-bottom: 0;
}

.count {
  min-width: 30px;
  height: 24px;
  display: inline-grid;
  place-items: center;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--inner);
  color: var(--muted);
  font-size: 12px;
}

.records-list {
  display: grid;
  align-content: start;
  gap: 10px;
  max-height: 42vh;
  overflow: auto;
  padding: 2px;
}

.record {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  padding: 11px 12px;
  border: 1px solid var(--line);
  border-left: 3px solid var(--accent);
  border-radius: 6px;
  background: var(--inner);
}

.record-index {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 6px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 13px;
  font-weight: 700;
}

.record-main {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.record-main strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.record-main span {
  color: var(--muted);
  font-size: 12px;
}

.empty {
  padding: 26px 0;
  border: 1px dashed var(--line-strong);
  border-radius: 6px;
  text-align: center;
}

.empty p {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
}

.activity-panel {
  flex: 1;
  min-height: 120px;
  overflow: auto;
  padding: 14px 16px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
}

.activity-list {
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.activity-list li {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
  color: var(--muted);
  font-size: 13px;
}

.activity-list time {
  flex: 0 0 auto;
  color: var(--accent);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.activity-list span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-li {
  color: var(--muted);
  font-size: 13px;
}

.capture-state {
  margin: 0 0 12px;
  color: var(--muted);
  font-size: 13px;
}

.color-field {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 10px;
  min-height: 40px;
}

.color-field span {
  color: var(--muted);
  font-size: 13px;
}

.color-field input[type="color"] {
  width: 52px;
  height: 32px;
  padding: 2px;
  border: 1px solid var(--line-strong);
  border-radius: 6px;
  background: var(--input);
  cursor: pointer;
}

.bullet-layer {
  position: fixed;
  inset: 68px 0 0 0;
  z-index: 80;
  pointer-events: none;
  overflow: hidden;
}

.bullet {
  position: absolute;
  left: 100%;
  display: inline-flex;
  align-items: center;
  gap: 12px;
  max-width: 86vw;
  padding: 8px 16px;
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 999px;
  background: rgba(8, 12, 16, 0.42);
  color: var(--bullet-color);
  white-space: nowrap;
  animation: bullet-fly 9s linear forwards;
  will-change: transform;
}

.bullet-group {
  flex: 0 0 auto;
  color: var(--bullet-color);
  font-style: normal;
  font-weight: 800;
  font-size: 22px;
  line-height: 1.2;
}

.bullet-content {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 15px;
  line-height: 1.3;
}

@keyframes bullet-fly {
  to {
    transform: translateX(calc(-100vw - 100%));
  }
}

.record-main .record-content {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.record-main .record-time {
  font-size: 11px;
}

.status-bar {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 42px;
  margin-top: 14px;
  padding: 0 14px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
  color: var(--text);
  font-size: 13px;
}

.status-dot {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--muted);
}

.status-bar.success .status-dot {
  background: var(--accent);
}

.status-bar.error .status-dot {
  background: var(--danger);
}

.status-bar.info .status-dot {
  background: var(--warn);
  animation: pulse 1.1s ease-in-out infinite;
}

@media (max-width: 760px) {
  .workspace {
    grid-template-columns: 1fr;
  }

  .records-list {
    max-height: 30vh;
  }
}
</style>
