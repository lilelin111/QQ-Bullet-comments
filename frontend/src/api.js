function appApi() {
  const root = window.go || window.__wails__?.go
  if (!root?.main?.App) {
    throw new Error('Wails 后端尚未就绪')
  }
  return root.main.App
}

function pickUser(payload) {
  const user = payload?.user
  if (!user) {
    return null
  }
  return {
    id: Number(user.ID ?? user.id ?? user.UserId ?? user.userId ?? 0),
    name: String(user.Name ?? user.name ?? ''),
  }
}

function normalize(result) {
  const success = Boolean(result?.success)
  const message = String(result?.message ?? (success ? '操作成功' : '操作失败'))
  return {
    success,
    message,
    user: pickUser(result),
    value: message,
    groupName: String(result?.group_name ?? result?.groupName ?? ''),
    messageId: Number(result?.message_id ?? result?.messageId ?? 0),
  }
}

function userPayload(user) {
  if (!user) {
    return {}
  }
  // 同时带上 Go 字段名和 JSON 标签名，避免序列化键名不一致
  return {
    ID: Number(user.id),
    id: Number(user.id),
    Name: String(user.name),
    name: String(user.name),
    Password: '',
    password: '',
  }
}

export async function register(username, password) {
  const result = await appApi().Register(username, password)
  return normalize(result)
}

export async function login(username, password) {
  const result = await appApi().Login(username, password)
  return normalize(result)
}

export async function createMessage(user) {
  const result = await appApi().CreateMessage(userPayload(user))
  return normalize(result)
}

export async function showGetTitle(userId, id) {
  const result = await appApi().ShowGetTitle(Number(userId), Number(id))
  return normalize(result)
}

export async function showGetMessage(userId, id) {
  const result = await appApi().ShowGetMessage(Number(userId), Number(id))
  return normalize(result)
}
