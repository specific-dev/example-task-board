import { useState, useEffect, useCallback } from "react"

const TOKEN_KEY = "auth_token"

export interface User {
  id: number
  email: string
  name: string
  avatar_url: string
}

export function useAuth(apiUrl: string) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [token, setToken] = useState<string | null>(() =>
    localStorage.getItem(TOKEN_KEY),
  )

  const saveToken = useCallback((t: string) => {
    localStorage.setItem(TOKEN_KEY, t)
    setToken(t)
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    setToken(null)
    setUser(null)
  }, [])

  useEffect(() => {
    if (!token) {
      setLoading(false)
      return
    }

    fetch(`${apiUrl}/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => {
        if (!res.ok) {
          logout()
          return null
        }
        return res.json()
      })
      .then((data) => {
        if (data) setUser(data)
      })
      .catch(() => logout())
      .finally(() => setLoading(false))
  }, [token, apiUrl, logout])

  const authFetch = useCallback(
    (url: string, opts: RequestInit = {}) => {
      return fetch(url, {
        ...opts,
        headers: {
          ...opts.headers,
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      })
    },
    [token],
  )

  return { user, loading, token, saveToken, logout, authFetch }
}
