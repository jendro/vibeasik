import type { ProjectComponent } from '~/types'

// Isian awal dialog tambah komponen menurut peran (backend, FE web/admin, mobile, folder lain).
export const defaultComponentPreset = (role: string, rootPath?: string): Partial<ProjectComponent> => {
  const root = rootPath || '~/Documents'
  const defaults: Record<string, Partial<ProjectComponent>> = {
    backend: {
      name: 'Backend API (BE)',
      role: 'backend',
      relative_path: 'BE',
      absolute_path: `${root}/BE`,
      language: 'go',
      framework: 'fiber',
      package_manager: 'go',
      dev_command: 'go run main.go',
      port: 8080,
      health_url: 'http://localhost:8080/health',
      env_file: '.env',
      env_summary: ''
    },
    frontend_client: {
      name: 'Frontend Web (FE)',
      role: 'frontend_client',
      relative_path: 'FE',
      absolute_path: `${root}/FE`,
      language: 'typescript',
      framework: 'nuxt',
      package_manager: 'bun',
      dev_command: 'bun run dev',
      port: 3000,
      health_url: 'http://localhost:3000',
      env_file: '.env',
      env_summary: ''
    },
    frontend_admin: {
      name: 'Admin Dashboard',
      role: 'frontend_admin',
      relative_path: 'admin',
      absolute_path: `${root}/admin`,
      language: 'typescript',
      framework: 'nuxt',
      package_manager: 'bun',
      dev_command: 'bun run dev',
      port: 3001,
      health_url: 'http://localhost:3001',
      env_file: '.env',
      env_summary: ''
    },
    frontend_mobile: {
      name: 'Mobile App',
      role: 'frontend_mobile',
      relative_path: 'mobile',
      absolute_path: `${root}/mobile`,
      language: 'dart',
      framework: 'flutter',
      package_manager: 'pub',
      dev_command: 'flutter run',
      port: 0,
      health_url: '',
      env_file: '.env',
      env_summary: ''
    }
  }

  return defaults[role] || {
    name: 'Folder Lain',
    role: role || 'other',
    relative_path: 'docs',
    absolute_path: `${root}/docs`,
    language: 'auto',
    port: 0
  }
}
