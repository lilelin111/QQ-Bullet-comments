<script setup>
import { reactive, ref } from 'vue'
import * as api from './api.js'

const mode = ref('login')
const credentials = reactive({
  username: '',
  password: '',
})
const currentUser = ref(null)
const capturing = ref(false)
const queryId = ref(0)
const queryResult = reactive({
  group: '',
  message: '',
})
const status = ref({ kind: 'idle', text: '就绪' })
const activity = ref([])
const records = ref([])
const nextRecordId = ref(1)

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
  capturing.value = false
  setStatus('idle', '已退出账号')
}

async function startCapture() {
  if (capturing.value) {
    return
  }

  capturing.value = true
  setStatus('info', '等待新的 QQ 通知…')

  try {
    const result = await api.createMessage(currentUser.value)
    if (!result.success) {
      setStatus('error', result.message)
      return
    }

    const groupName = result.groupName || 'QQ消息'
    records.value.unshift({
      id: nextRecordId.value++,
      group: groupName,
      time: nowText(),
    })
    pushActivity(`已保存弹幕：${groupName}`)
    setStatus('success', result.message)
  } catch (err) {
    setStatus('error', String(err?.message || err))
  } finally {
    capturing.value = false
  }
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
  <div class="app">
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
              <h2>捕获弹幕</h2>
              <span class="live-dot" :class="{ waiting: capturing }"></span>
            </div>
            <button
              class="primary-btn full"
              type="button"
              :disabled="capturing"
              @click="startCapture"
            >
              {{ capturing ? '等待 QQ 通知…' : '开始捕获' }}
            </button>
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
                <strong>{{ record.group }}</strong>
                <span>{{ record.time }}</span>
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
