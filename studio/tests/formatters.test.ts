import { describe, expect, test } from 'bun:test'
import {
  useFormatters,
  getCompRoleLabel,
  COMP_ROLE_LABELS
} from '../app/composables/useFormatters'

describe('useFormatters & getCompRoleLabel', () => {
  describe('COMP_ROLE_LABELS & getCompRoleLabel', () => {
    test('memetakan semua role yang terdaftar di COMP_ROLE_LABELS', () => {
      expect(getCompRoleLabel('backend')).toBe('Backend API')
      expect(getCompRoleLabel('frontend_admin')).toBe('Admin Dashboard')
      expect(getCompRoleLabel('frontend_client')).toBe('Frontend Web')
      expect(getCompRoleLabel('frontend_mobile')).toBe('Mobile App')
      expect(getCompRoleLabel('docs')).toBe('Dokumentasi')
      expect(getCompRoleLabel('cli')).toBe('CLI Tool')
      expect(getCompRoleLabel('worker')).toBe('Worker Service')
      expect(getCompRoleLabel('other')).toBe('Folder Lain')
    })

    test('tabel lookup COMP_ROLE_LABELS memiliki kunci lengkap', () => {
      expect(COMP_ROLE_LABELS['backend']).toBe('Backend API')
      expect(COMP_ROLE_LABELS['frontend_admin']).toBe('Admin Dashboard')
      expect(COMP_ROLE_LABELS['frontend_client']).toBe('Frontend Web')
      expect(COMP_ROLE_LABELS['frontend_mobile']).toBe('Mobile App')
      expect(COMP_ROLE_LABELS['docs']).toBe('Dokumentasi')
      expect(COMP_ROLE_LABELS['cli']).toBe('CLI Tool')
      expect(COMP_ROLE_LABELS['worker']).toBe('Worker Service')
      expect(COMP_ROLE_LABELS['other']).toBe('Folder Lain')
    })

    test('mengembalikan fallback role asli jika role tidak terdaftar di lookup table', () => {
      expect(getCompRoleLabel('custom_engine')).toBe('custom_engine')
      expect(getCompRoleLabel('microservice')).toBe('microservice')
    })

    test('mengembalikan string kosong bila role undefined atau kosong', () => {
      expect(getCompRoleLabel('')).toBe('')
      expect(getCompRoleLabel(undefined)).toBe('')
    })

    test('getCompRoleLabel dapat diakses dari instance composable useFormatters', () => {
      const formatters = useFormatters()
      expect(formatters.getCompRoleLabel('backend')).toBe('Backend API')
      expect(formatters.getCompRoleLabel('frontend_client')).toBe('Frontend Web')
      expect(formatters.getCompRoleLabel('other')).toBe('Folder Lain')
      expect(formatters.getCompRoleLabel('unknown')).toBe('unknown')
    })
  })

  describe('useFormatters helper methods', () => {
    const formatters = useFormatters()

    test('getRoleBadgeColor mengembalikan warna badge sesuai role', () => {
      expect(formatters.getRoleBadgeColor('backend')).toBe('info')
      expect(formatters.getRoleBadgeColor('frontend_admin')).toBe('secondary')
      expect(formatters.getRoleBadgeColor('frontend_client')).toBe('success')
      expect(formatters.getRoleBadgeColor('frontend_mobile')).toBe('warning')
      expect(formatters.getRoleBadgeColor('unknown')).toBe('neutral')
      expect(formatters.getRoleBadgeColor(undefined)).toBe('neutral')
    })

    test('getRoleLabel mengembalikan label ringkas', () => {
      expect(formatters.getRoleLabel('backend')).toBe('BE')
      expect(formatters.getRoleLabel('frontend_admin')).toBe('FE Admin')
      expect(formatters.getRoleLabel('frontend_client')).toBe('FE Web')
      expect(formatters.getRoleLabel('frontend_mobile')).toBe('Mobile')
      expect(formatters.getRoleLabel('frontend_mobile', { mobile: 'App' })).toBe('App')
      expect(formatters.getRoleLabel('other')).toBe('Lainnya')
      expect(formatters.getRoleLabel('other', { other: 'Lain' })).toBe('Lain')
    })

    test('getSeverityBadgeColor mengembalikan warna severity', () => {
      expect(formatters.getSeverityBadgeColor('CRITICAL')).toBe('error')
      expect(formatters.getSeverityBadgeColor('WARNING')).toBe('warning')
      expect(formatters.getSeverityBadgeColor('INFO')).toBe('info')
      expect(formatters.getSeverityBadgeColor('UNKNOWN')).toBe('neutral')
    })

    test('formatBytes memformat ukuran byte dengan benar', () => {
      expect(formatters.formatBytes(null)).toBe('0 B')
      expect(formatters.formatBytes(0)).toBe('0 B')
      expect(formatters.formatBytes(500)).toBe('500 B')
      expect(formatters.formatBytes(1024)).toBe('1 KB')
      expect(formatters.formatBytes(1024 * 1024 * 2.5)).toBe('2.5 MB')
    })

    test('getMethodBadgeColor mengembalikan warna badge HTTP method', () => {
      expect(formatters.getMethodBadgeColor('GET')).toBe('success')
      expect(formatters.getMethodBadgeColor('POST')).toBe('info')
      expect(formatters.getMethodBadgeColor('PUT')).toBe('warning')
      expect(formatters.getMethodBadgeColor('PATCH')).toBe('warning')
      expect(formatters.getMethodBadgeColor('DELETE')).toBe('error')
      expect(formatters.getMethodBadgeColor('PAGE_VIEW')).toBe('primary')
    })
  })
})
