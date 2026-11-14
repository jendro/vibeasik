// Tiga jenis tombol aksi di Studio, dibedakan dengan ikon & penanda yang sama di semua halaman:
//   - scan: memeriksa/mengukur kode atau aplikasi di laptop, tanpa LLM (audit statis, Deep Check, uji runtime, Big-O);
//   - process: mengolah temuan — menjadikannya todo, menyelesaikan, mengabaikan, memvonis;
//   - llm: memanggil LLM lewat API (memakai kuota/biaya penyedia).
// Aksi proses yang memanggil LLM memakai jenis process + penanda LLM, sehingga pengguna tahu mana yang berbiaya.
export type ActionKind = 'scan' | 'process' | 'llm'

export interface ActionKindMeta {
  icon: string
  label: string
  hint: string
}

export const ACTION_KINDS: Record<ActionKind, ActionKindMeta> = {
  scan: { icon: 'i-lucide-scan-search', label: 'Periksa', hint: 'Memeriksa kode/aplikasi di laptop ini, tanpa LLM' },
  process: { icon: 'i-lucide-list-checks', label: 'Proses temuan', hint: 'Mengolah temuan: jadikan todo, selesaikan, atau abaikan' },
  llm: { icon: 'i-lucide-sparkles', label: 'LLM', hint: 'Memanggil LLM lewat API — memakai kuota penyedia' }
}

// Teks tooltip tombol: penjelasan aksinya sendiri lalu jenisnya (dan bahwa ia memanggil LLM).
export const actionTitle = (kind: ActionKind, llm: boolean, desc?: string): string => {
  const parts = [desc, ACTION_KINDS[kind].hint]
  if (llm && kind !== 'llm') parts.push(ACTION_KINDS.llm.hint)
  return parts.filter(Boolean).join(' · ')
}
