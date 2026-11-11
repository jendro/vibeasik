<script setup lang="ts">
import type { Finding } from '~/types'
import type { DropdownMenuItem } from '@nuxt/ui'

const { modelName } = useModelNames()

const props = defineProps<{
  finding: Finding
  resolving: boolean
  hasCreatedTodo?: boolean
}>()

const emit = defineEmits<{
  (e: 'resolve', finding: Finding): void
  (e: 'reResolve', finding: Finding): void
  (e: 'viewResolution', finding: Finding): void
  (e: 'ignore', finding: Finding): void
  (e: 'delete', finding: Finding): void
  // Vonis pembelajaran audit: verify = cek ulang via LLM, clear = cabut vonis
  (e: 'feedback', finding: Finding, mode: 'verify' | 'false_positive' | 'confirmed' | 'clear'): void
}>()

const router = useRouter()
const toast = useToast()
const expanded = ref(false)
const copiedCode = ref(false)

const isResolved = computed(() => props.finding.status === 'resolved' || props.hasCreatedTodo)
const isIgnored = computed(() => props.finding.status === 'ignored')
const isFalsePositive = computed(() => props.finding.verdict === 'false_positive')
const isConfirmed = computed(() => props.finding.verdict === 'confirmed')
const hasSavedResolution = computed(() => !!(props.finding.action_steps?.length || props.finding.code_after || props.finding.placement_plan?.length || props.finding.resolution_title))

const copyCode = async (code?: string) => {
  if (!code) return
  try {
    await navigator.clipboard.writeText(code)
    copiedCode.value = true
    toast.add({
      title: 'Kode Disalin',
      description: 'Kode perbaikan berhasil disalin ke clipboard.',
      icon: 'i-lucide-check',
      color: 'success'
    })
    setTimeout(() => { copiedCode.value = false }, 2000)
  } catch (err) {
    console.error('Copy failed:', err)
  }
}

const goToTodos = () => {
  router.push('/todos')
}

const menuItems = computed<DropdownMenuItem[][]>(() => {
  const primaryGroup: DropdownMenuItem[] = [
    {
      label: 'Selesaikan Ulang via LLM',
      icon: 'i-lucide-sparkles',
      onSelect: () => emit('reResolve', props.finding)
    }
  ]

  if (props.finding.todo_id || isResolved.value) {
    primaryGroup.unshift({
      label: 'Lihat Solusi & Langkah TODO',
      icon: 'i-lucide-sparkles',
      onSelect: () => emit('viewResolution', props.finding)
    })
  }

  if (props.finding.todo_id) {
    primaryGroup.push({
      label: `Buka Agenda TODO #${props.finding.todo_id}`,
      icon: 'i-lucide-external-link',
      onSelect: () => goToTodos()
    })
  }

  primaryGroup.push({
    label: isIgnored.value ? 'Batal Abaikan Temuan' : 'Abaikan Temuan',
    icon: isIgnored.value ? 'i-lucide-eye' : 'i-lucide-eye-off',
    onSelect: () => emit('ignore', props.finding)
  })

  const learningGroup: DropdownMenuItem[] = [
    {
      label: 'Cek Ulang via LLM',
      icon: 'i-lucide-scan-search',
      onSelect: () => emit('feedback', props.finding, 'verify')
    }
  ]
  if (!isFalsePositive.value) {
    learningGroup.push({
      label: 'Tandai Audit Keliru (Kode Sudah Benar)',
      icon: 'i-lucide-shield-off',
      onSelect: () => emit('feedback', props.finding, 'false_positive')
    })
  }
  if (!isConfirmed.value) {
    learningGroup.push({
      label: 'Tandai Temuan Benar',
      icon: 'i-lucide-badge-check',
      onSelect: () => emit('feedback', props.finding, 'confirmed')
    })
  }
  if (props.finding.verdict) {
    learningGroup.push({
      label: 'Cabut Vonis Pembelajaran',
      icon: 'i-lucide-undo-2',
      onSelect: () => emit('feedback', props.finding, 'clear')
    })
  }

  return [
    primaryGroup,
    learningGroup,
    [
      {
        label: 'Hapus Temuan',
        icon: 'i-lucide-trash-2',
        color: 'error' as const,
        onSelect: () => emit('delete', props.finding)
      }
    ]
  ]
})

const { getSeverityBadgeColor } = useFormatters()

const { getCategoryIcon } = useFormatters()

const getCategoryColor = (cat: string) => {
  switch (cat?.toLowerCase()) {
    case 'query':
    case 'database':
      return 'text-violet-500'
    case 'router':
    case 'api':
      return 'text-blue-500'
    case 'code_smell':
      return 'text-amber-500'
    case 'security':
      return 'text-rose-500'
    case 'port':
      return 'text-teal-500'
    case 'runtime':
      return 'text-emerald-500'
    case 'frontend':
      return 'text-pink-500'
    case 'mobile':
      return 'text-sky-500'
    case 'deployment':
    case 'config':
      return 'text-orange-500'
    default:
      return 'text-neutral-500'
  }
}
</script>

<template>
  <div
    class="p-4 rounded-xl border transition-all bg-white dark:bg-[#04111f]"
    :class="[
      isIgnored ? 'opacity-70 border-neutral-200 dark:border-white/[0.08]'
      : finding.severity === 'CRITICAL'
        ? 'border-rose-200 dark:border-rose-900/60 hover:border-rose-300 dark:hover:border-rose-800'
        : finding.severity === 'WARNING'
          ? 'border-amber-200 dark:border-amber-900/60 hover:border-amber-300 dark:hover:border-amber-800'
          : 'border-neutral-200 dark:border-white/[0.08] hover:border-neutral-300 dark:hover:border-neutral-700'
    ]"
  >
    <!-- Card Summary (Calm & Clean Default State) -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="space-y-1.5 flex-1 min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <!-- Category Icon -->
          <UIcon :name="getCategoryIcon(finding.category)" class="w-4 h-4 shrink-0" :class="getCategoryColor(finding.category)" />

          <!-- Rule ID Tag -->
          <span
            v-if="finding.rule_id"
            class="px-2 py-0.5 rounded text-xs font-mono font-bold bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 border border-neutral-200 dark:border-neutral-700"
          >
            {{ finding.rule_id }}
          </span>

          <!-- Title -->
          <h3
            class="text-sm font-bold truncate"
            :class="isIgnored ? 'text-neutral-500 dark:text-neutral-400 line-through' : 'text-neutral-900 dark:text-white'"
          >
            {{ finding.title }}
          </h3>

          <!-- Severity Badge -->
          <UBadge size="xs" :color="getSeverityBadgeColor(finding.severity)" variant="subtle">
            {{ finding.severity }}
          </UBadge>

          <!-- Deep Check Badge -->
          <UBadge
            v-if="finding.is_deep || finding.rule_id?.startsWith('DEEP-') || finding.title?.startsWith('[DEEP]')"
            size="xs"
            color="primary"
            variant="solid"
            icon="i-lucide-layers"
            class="font-mono text-xs font-semibold px-1.5 py-0"
            title="Temuan ini dideteksi khusus melalui Deep Check"
          >
            DEEP
          </UBadge>

          <!-- LLM Review Badge -->
          <UBadge
            v-if="finding.is_llm || finding.rule_id?.startsWith('LLM-REVIEW-')"
            size="xs"
            color="secondary"
            variant="soft"
            icon="i-lucide-sparkles"
            class="font-mono text-xs font-semibold px-1.5 py-0"
            :title="`Temuan dari review audit LLM${finding.model_used ? ` (${modelName(finding.model_used)})` : ''} — verifikasi sebelum diterapkan`"
          >
            LLM
          </UBadge>

          <!-- Vonis Pembelajaran Audit -->
          <UBadge
            v-if="isFalsePositive"
            size="xs"
            color="warning"
            variant="subtle"
            icon="i-lucide-shield-off"
            :title="finding.verdict_reason ? `Audit keliru: ${finding.verdict_reason}` : 'Audit keliru — kode sudah benar'"
          >
            Audit Keliru
          </UBadge>
          <UBadge
            v-else-if="isConfirmed"
            size="xs"
            color="error"
            variant="soft"
            icon="i-lucide-badge-check"
            :title="finding.verdict_reason ? `Temuan benar: ${finding.verdict_reason}` : 'Temuan terkonfirmasi benar'"
          >
            Terkonfirmasi
          </UBadge>

          <!-- Resolved Indicator -->
          <UBadge v-if="isResolved" size="xs" color="success" variant="subtle" icon="i-lucide-check-circle">
            Diselesaikan via LLM
          </UBadge>

          <!-- Todo ID Pill -->
          <span
            v-if="finding.todo_id"
            class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-mono font-bold bg-sky-100 dark:bg-sky-950/60 text-sky-800 dark:text-sky-300 border border-sky-300 dark:border-sky-800 cursor-pointer hover:bg-sky-200 transition-colors"
            title="Klik untuk membuka daftar Todo"
            @click.stop="goToTodos"
          >
            <UIcon name="i-lucide-clipboard-check" class="w-3 h-3 text-sky-600 dark:text-sky-400" />
            TODO #{{ finding.todo_id }}
          </span>

          <!-- Ignored Indicator -->
          <UBadge v-else-if="isIgnored && !isFalsePositive" size="xs" color="neutral" variant="subtle" icon="i-lucide-eye-off">
            Diabaikan
          </UBadge>
        </div>

        <!-- Description -->
        <p class="text-xs text-neutral-600 dark:text-neutral-400 line-clamp-2">
          {{ finding.description }}
        </p>

        <!-- Target File & Line Pill -->
        <div v-if="finding.target_file" class="flex items-center gap-2 text-xs text-neutral-500 font-mono">
          <UIcon name="i-lucide-file-code" class="w-3.5 h-3.5 text-neutral-400" />
          <span class="truncate">{{ finding.target_file }}</span>
          <span v-if="finding.line" class="px-1.5 py-0.5 rounded bg-neutral-100 dark:bg-neutral-800 text-neutral-700 dark:text-neutral-300 font-bold">
            Baris {{ finding.target_lines || finding.line }}
          </span>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-1.5 shrink-0">
        <!-- Main LLM Resolve Button (Hidden by default when already resolved or ignored) -->
        <UButton
          v-if="!isResolved && !isIgnored"
          size="xs"
          color="primary"
          variant="solid"
          icon="i-lucide-sparkles"
          :loading="resolving"
          title="Selesaikan temuan baris ini saja via LLM dan buat rencana aksi TODO"
          @click="emit('resolve', finding)"
        >
          Selesaikan via LLM
        </UButton>

        <!-- View LLM Resolution Button (Visible when resolved) -->
        <UButton
          v-if="isResolved"
          size="xs"
          color="success"
          variant="subtle"
          icon="i-lucide-sparkles"
          title="Lihat solusi dan langkah penyelesaian LLM"
          @click="emit('viewResolution', finding)"
        >
          Lihat Solusi LLM
        </UButton>

        <!-- Titik 3 (Dropdown Menu: Selesaikan Ulang, Buka Todo, Abaikan, Delete) -->
        <UDropdownMenu :items="menuItems" size="xs">
          <UButton
            icon="i-lucide-more-vertical"
            color="neutral"
            variant="outline"
            size="xs"
            :loading="resolving"
            title="Menu Tindakan Temuan"
            aria-label="Menu Tindakan Temuan"
          />
        </UDropdownMenu>

        <!-- Button Saran Perbaikan (Uniform UButton size="xs") -->
        <UButton
          size="xs"
          color="neutral"
          variant="outline"
          :trailing-icon="expanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
          :title="expanded ? 'Tutup Detail Perbaikan' : 'Buka Detail Perbaikan'"
          aria-label="Detail Perbaikan"
          @click="expanded = !expanded"
        >
          {{ hasSavedResolution ? 'Detail Solusi & Langkah' : 'Saran Perbaikan' }}
        </UButton>
      </div>
    </div>

    <!-- Progressive Disclosure: In-Depth Solution, Steps & Code Context -->
    <div
      v-if="expanded"
      class="mt-4 pt-4 border-t border-neutral-100 dark:border-white/[0.08] space-y-4 duration-150"
    >
      <!-- Alasan vonis pembelajaran audit -->
      <div
        v-if="finding.verdict && finding.verdict_reason"
        class="p-3 rounded-xl border text-xs flex items-start gap-2"
        :class="isFalsePositive
          ? 'bg-amber-50/70 dark:bg-amber-950/20 border-amber-200 dark:border-amber-900/60'
          : 'bg-rose-50/70 dark:bg-rose-950/20 border-rose-200 dark:border-rose-900/60'"
      >
        <UIcon name="i-lucide-graduation-cap" class="w-4 h-4 shrink-0 mt-0.5" />
        <div>
          <span class="font-bold text-neutral-900 dark:text-white">
            {{ isFalsePositive ? 'Audit keliru — bahan pembaruan rule' : 'Temuan benar — pelajaran untuk agent coding' }}
          </span>
          <p class="text-neutral-700 dark:text-neutral-300 mt-0.5">
            {{ finding.verdict_reason }}
          </p>
        </div>
      </div>

      <!-- Case 1: Finding Has Full Saved LLM Resolution -->
      <template v-if="hasSavedResolution">
        <!-- Resolution Overview Banner -->
        <div class="p-3.5 rounded-xl bg-violet-50/70 dark:bg-violet-950/30 border border-violet-200 dark:border-violet-800/60 space-y-2 text-xs">
          <div class="flex items-center justify-between">
            <span class="font-bold text-sm text-neutral-900 dark:text-white">
              {{ finding.resolution_title || finding.title }}
            </span>
            <span v-if="finding.model_used" class="flex items-center gap-1 text-xs text-neutral-500 font-mono">
              <UIcon name="i-lucide-cpu" class="w-3.5 h-3.5" />
              {{ modelName(finding.model_used) }}
            </span>
          </div>

          <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed font-sans">
            {{ finding.explanation || finding.description }}
          </p>

          <!-- Theory Basis -->
          <div v-if="finding.theory_basis" class="flex items-start gap-1.5 pt-1 text-xs text-violet-700 dark:text-violet-300 font-medium">
            <UIcon name="i-lucide-book-open" class="w-4 h-4 shrink-0 mt-0.5" />
            <span>Landasan Teori: {{ finding.theory_basis }}</span>
          </div>
        </div>

        <!-- Todo Saved Confirmation Banner -->
        <div
          v-if="finding.todo_id"
          class="p-3 rounded-xl bg-sky-50 dark:bg-sky-950/30 border border-sky-200 dark:border-sky-800/60 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs"
        >
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-check-circle-2" class="w-4 h-4 text-sky-600 dark:text-sky-400 shrink-0" />
            <div>
              <span class="font-bold text-sky-900 dark:text-sky-300">
                Tersimpan di Modul Perencanaan (TODO #{{ finding.todo_id }})
              </span>
              <p class="text-xs text-sky-700 dark:text-sky-400 mt-0.5">
                AI Agent dapat langsung mengambil agenda ini dan mengeksekusi langkah-langkah di bawah.
              </p>
            </div>
          </div>
          <UButton size="xs" color="success" variant="outline" icon="i-lucide-external-link" @click="goToTodos">
            Buka Daftar TODO
          </UButton>
        </div>

        <!-- Action Steps for Agent / Dev -->
        <div v-if="finding.action_steps && finding.action_steps.length > 0" class="space-y-2">
          <div class="flex items-center gap-1.5 text-xs font-semibold text-neutral-700 dark:text-neutral-300">
            <UIcon name="i-lucide-list-ordered" class="w-3.5 h-3.5 text-primary-500" />
            <span>Langkah-langkah Penyelesaian (Action Steps untuk AI Agent):</span>
          </div>
          <div class="space-y-1.5">
            <div
              v-for="(step, idx) in finding.action_steps"
              :key="idx"
              class="flex items-start gap-2.5 p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/30 border border-neutral-200/80 dark:border-white/[0.08] text-xs text-neutral-800 dark:text-neutral-200 leading-relaxed"
            >
              <span class="w-5 h-5 rounded-full bg-primary-100 dark:bg-primary-950/60 text-primary-700 dark:text-primary-300 flex items-center justify-center font-bold text-xs shrink-0 mt-0.5 border border-primary-200 dark:border-primary-800">
                {{ idx + 1 }}
              </span>
              <span class="flex-1">{{ step }}</span>
            </div>
          </div>
        </div>

        <DiagnosticsPlacementPlanTable v-if="finding.placement_plan?.length" :plan="finding.placement_plan" />

        <!-- Code Replacement (Before vs After) -->
        <div v-if="finding.code_after" class="space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold text-neutral-700 dark:text-neutral-300 flex items-center gap-1.5">
              <UIcon name="i-lucide-code-2" class="w-3.5 h-3.5 text-sky-400" />
              Kode Rekomendasi Solusi (Drop-in Replacement):
            </span>
            <UButton
              size="xs"
              color="neutral"
              variant="outline"
              :icon="copiedCode ? 'i-lucide-check' : 'i-lucide-copy'"
              @click="copyCode(finding.code_after)"
            >
              {{ copiedCode ? 'Tersalin' : 'Salin Kode' }}
            </UButton>
          </div>

          <pre class="p-3 rounded-xl bg-[#010810] text-sky-300 font-mono text-xs overflow-x-auto border border-white/10 leading-relaxed max-h-60"><code>{{ finding.code_after }}</code></pre>
        </div>
      </template>

      <!-- Case 2: Standard Unresolved View (Theory Box, Snippet, Suggestion) -->
      <template v-else>
        <!-- Theory Box -->
        <div v-if="finding.theory_basis" class="p-3 rounded-lg bg-violet-50/60 dark:bg-violet-950/20 border border-violet-200/80 dark:border-violet-900/40 text-xs space-y-1">
          <div class="flex items-center gap-1.5 font-bold text-violet-800 dark:text-violet-300">
            <UIcon name="i-lucide-book-open" class="w-4 h-4 text-violet-600 dark:text-violet-400" />
            Landasan Teori & Standar Rekayasa Perangkat Lunak:
          </div>
          <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed font-sans pl-5">
            {{ finding.theory_basis }}
          </p>
        </div>

        <!-- Code Snippet -->
        <div v-if="finding.snippet" class="space-y-1">
          <span class="text-xs font-semibold text-neutral-500 dark:text-neutral-400">
            Kode Terindikasi (Baris {{ finding.line || '1' }}):
          </span>
          <pre class="p-2.5 rounded-lg bg-neutral-900 text-rose-300 font-mono text-xs overflow-x-auto border border-neutral-800"><code>{{ finding.snippet }}</code></pre>
        </div>

        <!-- Suggested Fix / Action -->
        <div class="space-y-1">
          <span class="text-xs font-semibold text-neutral-500 dark:text-neutral-400">
            Saran Perbaikan:
          </span>
          <div class="p-2.5 rounded-lg bg-neutral-50 dark:bg-neutral-800/40 border border-neutral-200 dark:border-white/[0.08] text-xs text-neutral-700 dark:text-neutral-300 font-mono leading-relaxed">
            {{ finding.suggestion }}
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
