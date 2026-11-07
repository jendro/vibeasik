// Satu sumber logika apiBase untuk semua composable API domain.
export const useApiBase = (): string => {
  const config = useRuntimeConfig()
  let apiBase = (config.public.apiBase as string) || 'http://localhost:8765'

  // Dynamic origin adaptation:
  // When served directly by vibeasik local-engine (e.g. http://localhost:8765 or LAN IP:port),
  // automatically use window.location.origin so all API calls match current host and port.
  // When running in the Nuxt dev server (any port), adapt LAN IP while targeting the engine port.
  if (import.meta.client && typeof window !== 'undefined') {
    if (!import.meta.dev) {
      apiBase = window.location.origin
    } else {
      try {
        const parsed = new URL(apiBase)
        if (
          (parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1')
          && window.location.hostname !== 'localhost'
          && window.location.hostname !== '127.0.0.1'
        ) {
          parsed.hostname = window.location.hostname
          apiBase = parsed.origin
        }
      } catch {
        // Keep original apiBase if URL parsing fails
      }
    }
  }

  return apiBase
}
