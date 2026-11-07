export interface AuthUser {
  id: number
  name: string
  email: string
  role: string
  avatar?: string
  toolPreference?: string
}

export const useAuth = () => {
  const router = useRouter()
  const user = useCookie<AuthUser | null>('vibeasik_auth_user', {
    maxAge: 60 * 60 * 24 * 30,
    path: '/',
    sameSite: 'lax'
  })
  const token = useCookie<string | null>('vibeasik_auth_token', {
    maxAge: 60 * 60 * 24 * 30,
    path: '/',
    sameSite: 'lax'
  })

  const isAuthenticated = computed(() => Boolean(token.value && user.value))

  const login = async (email: string, password: string): Promise<{ success: boolean, error?: string }> => {
    if (!email || !password) {
      return { success: false, error: 'Email dan password wajib diisi.' }
    }
    if (password.length < 4) {
      return { success: false, error: 'Password minimal 4 karakter.' }
    }

    // Set authenticated session
    token.value = `vibe_tok_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`
    user.value = {
      id: 1,
      name: email.split('@')[0] || 'Vibecoder',
      email: email,
      role: 'vibecoder',
      toolPreference: 'Antigravity / Cursor'
    }

    return { success: true }
  }

  const register = async (name: string, email: string, password: string, toolPreference = 'Antigravity / Cursor'): Promise<{ success: boolean, error?: string }> => {
    if (!name || !email || !password) {
      return { success: false, error: 'Nama, email, dan password wajib diisi.' }
    }
    if (password.length < 4) {
      return { success: false, error: 'Password minimal 4 karakter.' }
    }

    token.value = `vibe_tok_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`
    user.value = {
      id: Math.floor(Math.random() * 1000) + 2,
      name,
      email,
      role: 'vibecoder',
      toolPreference
    }

    return { success: true }
  }

  const quickDemoLogin = () => {
    token.value = `vibe_tok_demo_${Date.now()}`
    user.value = {
      id: 1,
      name: 'Vibecoder Developer',
      email: 'demo@vibeasik.com',
      role: 'vibecoder',
      toolPreference: 'Google Antigravity & Claude'
    }
    return { success: true }
  }

  const logout = () => {
    token.value = null
    user.value = null
    router.push('/')
  }

  return {
    user,
    token,
    isAuthenticated,
    login,
    register,
    quickDemoLogin,
    logout
  }
}
