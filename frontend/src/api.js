function appApi() {
  const root = window.go || window.__wails__?.go
  if (!root?.main?.App) {
    throw new Error('Wails 后端尚未就绪')
  }
  return root.main.App
}

function publicUser(value) {
  if (!value) {
    return null
  }

  return {
    id: Number(value.id ?? value.ID ?? value.userId ?? value.UserId ?? 0),
    name: String(value.name ?? value.Name ?? ''),
  }
}

function normalize(result) {
  const success = Boolean(result?.success)
  return {
    success,
    message: String(result?.message ?? (success ? '操作成功' : '操作失败')),
    user: publicUser(result?.user),
  }
}

export async function register(username, password) {
  return normalize(await appApi().Register(username, password))
}

export async function login(username, password) {
  return normalize(await appApi().Login(username, password))
}

export async function logout() {
  return normalize(await appApi().Logout())
}
