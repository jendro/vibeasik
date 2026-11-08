import { describe, expect, test } from 'bun:test'
import {
  addDynamicMeta,
  removeDynamicMeta,
  createDefaultExtractedEnv,
  ensureDynamicMetaContainer
} from '../app/utils/fastScanDynamicMeta'
import type { FastScanResult } from '../app/types'

const createMockScanResult = (overrides?: Partial<FastScanResult>): FastScanResult => ({
  component_id: 1,
  component_name: 'api',
  role: 'backend',
  relative_path: 'backend',
  absolute_path: '/path/to/backend',
  directory_tree: '',
  folder_structure_summary: '',
  how_to_run: { commands: [] },
  extracted_env: {
    database_type: 'postgres',
    orm_driver: 'prisma',
    database_name: 'app_db',
    auth_methods: ['jwt'],
    login_endpoints: ['/api/auth/login'],
    third_party_services: [],
    environment_keys: [],
    login_and_access_guide: '',
    dynamic_metadata: {
      redis_port: '6379'
    }
  },
  detected_params: {},
  ai_analysis_markdown: '',
  security_audit: {
    is_critical: false,
    redacted_count: 0,
    sensitive_keys: []
  },
  scanned_at: '2026-11-08T12:00:00Z',
  ...overrides
})

describe('fastScanDynamicMeta helper', () => {
  describe('createDefaultExtractedEnv', () => {
    test('membuat struktur default ExtractedEnvInfo yang lengkap', () => {
      const def = createDefaultExtractedEnv()
      expect(def.database_type).toBe('')
      expect(def.orm_driver).toBe('')
      expect(def.auth_methods).toEqual([])
      expect(def.dynamic_metadata).toEqual({})
    })
  })

  describe('ensureDynamicMetaContainer', () => {
    test('menginisialisasi extracted_env jika sebelumnya undefined', () => {
      const scanResult = createMockScanResult()
      // @ts-expect-error testing undefined extracted_env
      delete scanResult.extracted_env

      const container = ensureDynamicMetaContainer(scanResult)
      expect(container).toBeDefined()
      expect(scanResult.extracted_env).toBeDefined()
      expect(scanResult.extracted_env?.dynamic_metadata).toBeDefined()
    })

    test('menginisialisasi dynamic_metadata jika extracted_env ada tetapi dynamic_metadata undefined', () => {
      const scanResult = createMockScanResult()
      // @ts-expect-error testing undefined dynamic_metadata
      delete scanResult.extracted_env.dynamic_metadata

      const container = ensureDynamicMetaContainer(scanResult)
      expect(container).toBeDefined()
      expect(scanResult.extracted_env.dynamic_metadata).toEqual({})
    })
  })

  describe('addDynamicMeta', () => {
    test('mengembalikan false bila scanResult null atau undefined', () => {
      let errorMsg = ''
      const resNull = addDynamicMeta(null, 'key', 'val', (err) => { errorMsg = err })
      expect(resNull).toBe(false)
      expect(errorMsg).toBe('')

      const resUndef = addDynamicMeta(undefined, 'key', 'val')
      expect(resUndef).toBe(false)
    })

    test('validasi gagal bila key hanya whitespace atau kosong (callback function)', () => {
      const scanResult = createMockScanResult()
      let errorMsg = ''
      const res = addDynamicMeta(scanResult, '   ', 'some-value', (err) => {
        errorMsg = err
      })

      expect(res).toBe(false)
      expect(errorMsg).toBe('Nama properti / atribut wajib diisi')
      expect(scanResult.extracted_env.dynamic_metadata['   ']).toBeUndefined()
    })

    test('validasi gagal bila key kosong menggunakan options object { onError }', () => {
      const scanResult = createMockScanResult()
      let errorMsg = ''
      const res = addDynamicMeta(scanResult, '', 'val', {
        onError: (err) => { errorMsg = err }
      })

      expect(res).toBe(false)
      expect(errorMsg).toBe('Nama properti / atribut wajib diisi')
    })

    test('berhasil menambahkan metadata dengan key dan val yang di-trim', () => {
      const scanResult = createMockScanResult()
      const res = addDynamicMeta(scanResult, '  queue_driver  ', '  rabbitmq  ')

      expect(res).toBe(true)
      expect(scanResult.extracted_env.dynamic_metadata['queue_driver']).toBe('rabbitmq')
    })

    test('berhasil menginisialisasi extracted_env jika scanResult belum memilikinya', () => {
      const scanResult = createMockScanResult()
      // @ts-expect-error testing undefined extracted_env
      delete scanResult.extracted_env

      const res = addDynamicMeta(scanResult, 'grpc_port', '50051')
      expect(res).toBe(true)
      expect(scanResult.extracted_env.dynamic_metadata['grpc_port']).toBe('50051')
    })
  })

  describe('removeDynamicMeta', () => {
    test('mengembalikan false bila scanResult atau dynamic_metadata tidak ada', () => {
      expect(removeDynamicMeta(null, 'key')).toBe(false)
      expect(removeDynamicMeta(undefined, 'key')).toBe(false)

      const scanResult = createMockScanResult()
      // @ts-expect-error testing undefined extracted_env
      delete scanResult.extracted_env
      expect(removeDynamicMeta(scanResult, 'key')).toBe(false)
    })

    test('berhasil menghapus key yang ada di dynamic_metadata', () => {
      const scanResult = createMockScanResult()
      expect(scanResult.extracted_env.dynamic_metadata['redis_port']).toBe('6379')

      const res = removeDynamicMeta(scanResult, 'redis_port')
      expect(res).toBe(true)
      expect(scanResult.extracted_env.dynamic_metadata['redis_port']).toBeUndefined()
    })
  })
})
