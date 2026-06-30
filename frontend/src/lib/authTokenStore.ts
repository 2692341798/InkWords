const AUTH_TOKEN_STORAGE_KEY = 'token'
const AUTH_TOKEN_CHANGE_EVENT = 'inkwords:auth-token-changed'

const getStorage = () => {
  if (typeof window === 'undefined' && typeof globalThis.localStorage === 'undefined') {
    return null
  }
  return globalThis.localStorage ?? null
}

const eventTarget: EventTarget = typeof window === 'undefined' ? new EventTarget() : window

const notifyTokenChanged = () => {
  eventTarget.dispatchEvent(new Event(AUTH_TOKEN_CHANGE_EVENT))
}

const getSnapshot = () => {
  return getStorage()?.getItem(AUTH_TOKEN_STORAGE_KEY) ?? null
}

const getServerSnapshot = () => null

const setToken = (token: string) => {
  getStorage()?.setItem(AUTH_TOKEN_STORAGE_KEY, token)
  notifyTokenChanged()
}

const clearToken = () => {
  getStorage()?.removeItem(AUTH_TOKEN_STORAGE_KEY)
  notifyTokenChanged()
}

const subscribe = (listener: () => void) => {
  const onTokenChange = () => listener()
  const onStorage = (event: StorageEvent) => {
    if (event.key === AUTH_TOKEN_STORAGE_KEY) {
      listener()
    }
  }

  eventTarget.addEventListener(AUTH_TOKEN_CHANGE_EVENT, onTokenChange)
  if (typeof window !== 'undefined') {
    window.addEventListener('storage', onStorage)
  }

  return () => {
    eventTarget.removeEventListener(AUTH_TOKEN_CHANGE_EVENT, onTokenChange)
    if (typeof window !== 'undefined') {
      window.removeEventListener('storage', onStorage)
    }
  }
}

export const authTokenStore = {
  getSnapshot,
  getServerSnapshot,
  subscribe,
  setToken,
  clearToken,
}
