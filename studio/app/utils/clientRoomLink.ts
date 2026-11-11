// Tautan ke Ruang Klien (satu pintu permintaan) yang membawa niat & fokus: proyek, aplikasi, dan halaman/rute.
// Proyek ikut dibawa supaya Ruang Klien berpindah ke proyek pemilik aplikasi itu bila proyek aktifnya berbeda.
type ClientRoomFocus = { projectId?: number, componentId?: number, page?: string, kind?: string }

export const clientRoomLink = (focus: ClientRoomFocus) => ({
  path: '/client-room',
  query: {
    ...(focus.kind ? { kind: focus.kind } : {}),
    ...(focus.projectId ? { project: String(focus.projectId) } : {}),
    ...(focus.componentId ? { app: String(focus.componentId) } : {}),
    ...(focus.page ? { page: focus.page } : {})
  }
})

// Halaman antarmuka → rombak desain; rute backend → tambah/ubah fitur.
export const clientRoomRouteLink = (projectId: number | undefined, componentId: number | undefined, path: string, isFrontend: boolean) =>
  clientRoomLink({ projectId, componentId, page: path, kind: isFrontend ? 'design' : 'change' })
