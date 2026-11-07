<script setup lang="ts">
definePageMeta({
  public: true
})

// Cara memasang vibeasik: satu binary lokal (tanpa aplikasi desktop), dipakai lewat browser & MCP.
const installCmd = 'curl -fsSL https://vibeasik.com/install.sh | sh'
const copied = ref(false)
const toast = useToast()

const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = true
    setTimeout(() => (copied.value = false), 1800)
  } catch {
    toast.add({ title: 'Salin manual', description: text, color: 'neutral' })
  }
}

const steps = [
  { cmd: installCmd, note: 'Linux & macOS: unduh binary, verifikasi checksum, pasang ke ~/.local/bin.' },
  { cmd: 'vibeasik serve', note: 'Jalankan engine lokal (hanya localhost, dilindungi token).' },
  { cmd: 'vibeasik open', note: 'Buka Studio di browser dan masuk otomatis.' },
  { cmd: 'vibeasik mcp --help', note: 'Hubungkan agent coding (Claude Code, Cursor, Antigravity, Aider) lewat MCP.' },
  { cmd: 'vibeasik login', note: 'Opsional: masuk akun gratis untuk Pusat Pengetahuan; paket Pro membuka fitur berbayar.' }
]

useHead({ title: 'Pasang vibeasik' })
</script>

<template>
  <div class="max-w-3xl mx-auto py-10 px-4 space-y-8">
    <div class="space-y-2">
      <UBadge color="primary" variant="subtle" icon="i-lucide-terminal">
        Satu binary, jalan di laptop Anda
      </UBadge>
      <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">
        Pasang vibeasik
      </h1>
      <p class="text-neutral-600 dark:text-neutral-400">
        vibeasik berjalan lokal: kode Anda tidak diunggah. Dipakai lewat browser (Studio) dan agent coding (MCP).
      </p>
    </div>

    <ol class="space-y-4">
      <li v-for="(s, i) in steps" :key="i" class="rounded-lg border border-neutral-200 dark:border-white/10 p-4 space-y-2">
        <div class="flex items-center justify-between gap-2">
          <code class="font-mono text-sm break-all text-sky-700 dark:text-sky-300">{{ s.cmd }}</code>
          <UButton size="xs" variant="ghost" color="neutral" :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'" @click="copy(s.cmd)" />
        </div>
        <p class="text-xs text-neutral-500 dark:text-neutral-400">
          {{ s.note }}
        </p>
      </li>
    </ol>

    <div class="rounded-lg bg-neutral-50 dark:bg-neutral-900/50 p-4 text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
      <p><b>Windows:</b> unduh berkas <code>.zip</code> dari halaman Releases, ekstrak, lalu jalankan <code>vibeasik.exe serve</code>.</p>
      <p><b>Keamanan:</b> engine hanya menerima koneksi dari komputer ini dan meminta token lokal. Detail di SECURITY.md.</p>
    </div>
  </div>
</template>
