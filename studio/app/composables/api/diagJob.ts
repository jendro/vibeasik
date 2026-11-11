// Pekerjaan diag di antrean engine (Selesaikan temuan via LLM, Deep Check, vonis, satukan todo): engine menjawab
// 202 + job_id karena prosesnya bisa lebih lama dari batas waktu request. Fungsi ini menanyakan status job sampai
// selesai lalu mengembalikan hasil sebenarnya, atau melempar galat yang dilaporkan engine.
export const pollDiagJob = async <T>(apiBase: string, accepted: DiagJobAccepted | T, projectId?: number, timeoutMs = 10 * 60_000): Promise<T> => {
  const job = accepted as DiagJobAccepted
  if (!job || job.status !== 'processing' || !job.job_id) return accepted as T
  const pid = job.project_id ?? projectId
  const deadline = Date.now() + timeoutMs
  let delay = 1000
  while (Date.now() < deadline) {
    await new Promise(r => setTimeout(r, delay))
    delay = Math.min(delay + 500, 3000)
    const st = await $fetch<DiagJobState<T>>(`${apiBase}/api/projects/${pid}/diagnostics/jobs/${job.job_id}`)
    if (st.status === 'done') return st.result as T
    if (st.status === 'error') throw new Error(st.error || 'Pemrosesan di engine gagal.')
  }
  throw new Error(`Engine belum selesai memproses setelah ${Math.round(timeoutMs / 60_000)} menit; cek lagi nanti.`)
}
