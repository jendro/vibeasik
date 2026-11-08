import type { ExtractedEnvInfo, FastScanResult } from '~/types'

/**
 * Membuat struktur default untuk ExtractedEnvInfo jika belum ada.
 */
export const createDefaultExtractedEnv = (): ExtractedEnvInfo => ({
  database_type: '',
  orm_driver: '',
  database_name: '',
  auth_methods: [],
  login_endpoints: [],
  third_party_services: [],
  environment_keys: [],
  login_and_access_guide: '',
  dynamic_metadata: {}
})

/**
 * Memastikan extracted_env dan dynamic_metadata pada scanResult telah terinisialisasi.
 */
export const ensureDynamicMetaContainer = (scanResult: FastScanResult): Record<string, string> => {
  if (!scanResult.extracted_env) {
    scanResult.extracted_env = createDefaultExtractedEnv()
  }
  if (!scanResult.extracted_env.dynamic_metadata) {
    scanResult.extracted_env.dynamic_metadata = {}
  }
  return scanResult.extracted_env.dynamic_metadata
}

export type DynamicMetaErrorHandler = (description: string) => void

export interface AddDynamicMetaOptions {
  onError?: DynamicMetaErrorHandler
}

/**
 * Menambahkan metadata dinamis (key/value) ke dalam scanResult.
 * Mengembalikan true jika penambahan berhasil, atau false jika validasi gagal.
 */
export const addDynamicMeta = (
  scanResult: FastScanResult | null | undefined,
  key: string,
  value: string,
  options?: AddDynamicMetaOptions | DynamicMetaErrorHandler
): boolean => {
  if (!scanResult) return false

  const trimmedKey = key.trim()
  const trimmedVal = value.trim()

  if (!trimmedKey) {
    const errorHandler = typeof options === 'function' ? options : options?.onError
    if (errorHandler) {
      errorHandler('Nama properti / atribut wajib diisi')
    }
    return false
  }

  const dynamicMetadata = ensureDynamicMetaContainer(scanResult)
  dynamicMetadata[trimmedKey] = trimmedVal
  return true
}

/**
 * Menghapus entri metadata dinamis dari scanResult berdasarkan key.
 */
export const removeDynamicMeta = (
  scanResult: FastScanResult | null | undefined,
  key: string
): boolean => {
  if (!scanResult?.extracted_env?.dynamic_metadata) return false
  Reflect.deleteProperty(scanResult.extracted_env.dynamic_metadata, key)
  return true
}
