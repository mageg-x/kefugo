import { defineStore } from 'pinia'

const TOKEN_KEY = 'token'
const USER_KEY = 'user'

function sanitizeUser(user) {
  if (!user || typeof user !== 'object') {
    return null
  }
  return {
    id: user.id || user.ID || 0,
    username: user.username || user.name || '',
    name: user.username || user.name || '',
    role: user.role || '',
    avatar: user.avatar || '',
    email: user.email || '',
    phone: user.phone || '',
    status: Number(user.status || 0),
    active: Boolean(user.active),
  }
}

export const useStore = defineStore('global', {
  state: () => ({
    token: localStorage.getItem(TOKEN_KEY) || null,
    user: sanitizeUser(JSON.parse(localStorage.getItem(USER_KEY) || 'null'))
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
    isAdmin: (state) => state.user?.role === 'admin'
  },
  actions: {
    setUser(token, user) {
      const safeUser = sanitizeUser(user)
      this.token = token
      this.user = safeUser
      localStorage.setItem(TOKEN_KEY, token || '')
      localStorage.setItem(USER_KEY, JSON.stringify(safeUser))
    },
    clearUser() {
      this.token = null
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
    },
    reset() {
      this.token = null
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
    }
  }
})
