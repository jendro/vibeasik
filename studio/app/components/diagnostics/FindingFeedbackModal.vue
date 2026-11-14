<script setup lang="ts">
import type { AuditVerdict, Finding, FindingVerification } from '~/types'

const { modelName } = useModelNames()

// Modal vonis pembelajaran audit: cek ulang temuan via LLM, lalu manusia memutuskan
// "audit keliru" (kode sudah benar → rule perlu update) atau "temuan benar".
const open = defineModel<boolean>('open', { default: false })

const props = defineProps<{
  projectId: number
  finding: Finding | null
  initialVerdict?: AuditVerdict
  autoVerify?: boolean
}>()

const emit = defineEmits<{
  (e: 'saved', payload: { findings: Finding[], score: number, verdict: AuditVerdict }): void
}>()

const toast = useToast()
const { verifyDiagnosticFinding, recordFindingFeedback } = useApi()

const verdict = ref<AuditVerdict>('false_positive')
const reason = ref('')
const ruleAdvice = ref('')
const verifying = ref(false)
const saving = ref(false)
const verification = ref<FindingVerification | null>(null)
const verifyError = ref('')

const verdictOptions = [
  { value: 'false_positive', label: 'Audit keliru', desc: 'Kode sudah benar — temuan diabaikan, temuan serupa diredam, rule ditandai perlu update', icon: 'i-lucide-shield-off' },
  { value: 'confirmed', label: 'Temuan benar', desc: 'Masalah nyata — jadi pelajaran untuk agent coding & reviewer LLM', icon: 'i-lucide-badge-check' }
] as const

const runVerify = async () => {
  if (!props.finding || !props.projectId) return
  verifying.value = true
  verifyError.value = ''
  try {
    const res = await verifyDiagnosticFinding(props.projectId, props.finding)
    verification.value = res
    if (res.verdict !== 'uncertain') verdict.value = res.verdict
    if (!reason.value.trim()) reason.value = res.reason
    if (!ruleAdvice.value.trim() && res.rule_advice) ruleAdvice.value = res.rule_advice
  } catch (err: any) {
    verifyError.value = err?.data?.error || err?.message || 'Gagal memeriksa ulang via LLM'
  } finally {
    verifying.value = false
  }
}

watch(open, (isOpen) => {
  if (!isOpen) return
  verdict.value = props.initialVerdict || (props.finding?.verdict as AuditVerdict) || 'false_positive'
  reason.value = props.finding?.verdict_reason || ''
  ruleAdvice.value = ''
  verification.value = null
  verifyError.value = ''
  if (props.autoVerify) runVerify()
})

const verificationLabel = computed(() => {
  switch (verification.value?.verdict) {
    case 'false_positive': return 'LLM menilai audit keliru'
    case 'confirmed': return 'LLM menilai temuan benar'
    default: return 'LLM belum yakin'
  }
})

const save = async () => {
  if (!props.finding || !props.projectId) return
  if (!reason.value.trim()) {
    toast.add({ title: 'Alasan wajib diisi', description: 'Alasan inilah yang menjadi bahan pembelajaran audit.', color: 'warning', icon: 'i-lucide-info' })
    return
  }
  saving.value = true
  try {
    const usedLLM = !!verification.value && verification.value.verdict === verdict.value
    const res = await recordFindingFeedback(props.projectId, props.finding, {
      verdict: verdict.value,
      reason: reason.value.trim(),
      rule_advice: verdict.value === 'false_positive' ? ruleAdvice.value.trim() : '',
      source: usedLLM ? 'llm' : 'manual',
      model: usedLLM ? verification.value?.model : ''
    })
    emit('saved', { findings: res.findings, score: res.score, verdict: verdict.value })
    toast.add({
      title: verdict.value === 'false_positive' ? 'Ditandai audit keliru' : 'Ditandai temuan benar',
      description: verdict.value === 'false_positive'
        ? 'Temuan diabaikan, temuan serupa akan diredam otomatis, dan rule masuk daftar perlu update.'
        : 'Temuan menjadi pelajaran untuk agent coding & reviewer LLM berikutnya.',
      color: 'success',
      icon: 'i-lucide-graduation-cap'
    })
    open.value = false
  } catch (err: any) {
    toast.add({ title: 'Gagal menyimpan vonis', description: err?.data?.error || err?.message, color: 'error', icon: 'i-lucide-alert-circle' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Vonis Pembelajaran Audit"
    description="Tandai apakah temuan ini benar atau audit yang keliru. Vonis disimpan sebagai bahan pembelajaran audit & agent coding."
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <div v-if="finding" class="space-y-4 text-xs">
        <!-- Ringkasan temuan -->
        <div class="p-3 rounded-xl border border-neutral-200 dark:border-white/[0.08] bg-neutral-50 dark:bg-neutral-900/60 space-y-1.5">
          <div class="flex flex-wrap items-center gap-2">
            <span v-if="finding.rule_id" class="px-2 py-0.5 rounded text-xs font-mono font-bold bg-neutral-100 dark:bg-neutral-800 border border-neutral-200 dark:border-neutral-700">
              {{ finding.rule_id }}
            </span>
            <span class="font-bold text-sm text-neutral-900 dark:text-white">{{ finding.title }}</span>
          </div>
          <p class="text-neutral-600 dark:text-neutral-400">
            {{ finding.description }}
          </p>
          <p v-if="finding.target_file" class="font-mono text-xs text-neutral-500">
            {{ finding.target_file }}<span v-if="finding.line">:{{ finding.line }}</span>
          </p>
          <pre v-if="finding.snippet" class="p-2 rounded-lg bg-neutral-900 text-rose-300 font-mono overflow-x-auto"><code>{{ finding.snippet }}</code></pre>
        </div>

        <!-- Cek ulang via LLM -->
        <div class="p-3 rounded-xl border border-violet-200 dark:border-violet-900/60 bg-violet-50/60 dark:bg-violet-950/20 space-y-2">
          <div class="flex items-center justify-between gap-2">
            <span class="font-bold text-violet-800 dark:text-violet-300 flex items-center gap-1.5">
              <UIcon name="i-lucide-sparkles" class="w-4 h-4" />
              Cek ulang via LLM terhadap kode aktual
            </span>
            <AppActionButton
              kind="llm"
              size="xs"
              :label="verification ? 'Cek ulang lagi' : 'Cek ulang'"
              desc="LLM membaca kode di sekitar baris temuan dan menilai temuannya"
              :loading="verifying"
              @click="runVerify"
            />
          </div>
          <p v-if="verifying" class="text-neutral-500">
            LLM sedang membaca kode di sekitar baris temuan...
          </p>
          <p v-else-if="verifyError" class="text-rose-600 dark:text-rose-400">
            {{ verifyError }}
          </p>
          <div v-else-if="verification" class="space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <UBadge
                size="xs"
                :color="verification.verdict === 'false_positive' ? 'warning' : verification.verdict === 'confirmed' ? 'error' : 'neutral'"
                variant="subtle"
              >
                {{ verificationLabel }}
              </UBadge>
              <span class="text-xs text-neutral-500 font-mono">yakin {{ verification.confidence }}% · {{ modelName(verification.model) }}</span>
            </div>
            <p class="text-neutral-700 dark:text-neutral-300 leading-relaxed">
              {{ verification.reason }}
            </p>
          </div>
          <p v-else class="text-neutral-500">
            Opsional — hasil LLM hanya saran dan mengisi alasan otomatis; keputusan tetap di tangan Anda.
          </p>
        </div>

        <!-- Pilihan vonis -->
        <div class="grid sm:grid-cols-2 gap-2">
          <button
            v-for="opt in verdictOptions"
            :key="opt.value"
            type="button"
            class="text-left p-3 rounded-xl border transition-colors"
            :class="verdict === opt.value
              ? 'border-primary-400 dark:border-primary-600 bg-primary-50/70 dark:bg-primary-950/30'
              : 'border-neutral-200 dark:border-white/[0.08] hover:border-neutral-300 dark:hover:border-neutral-700'"
            @click="verdict = opt.value"
          >
            <span class="flex items-center gap-1.5 font-bold text-neutral-900 dark:text-white">
              <UIcon :name="opt.icon" class="w-4 h-4" />
              {{ opt.label }}
            </span>
            <span class="block mt-1 text-xs text-neutral-500">{{ opt.desc }}</span>
          </button>
        </div>

        <UFormField label="Alasan (wajib)" :description="verdict === 'false_positive' ? 'Mengapa kode ini sudah benar? Rujuk baris/pola kodenya.' : 'Mengapa ini masalah nyata?'">
          <UTextarea v-model="reason" :rows="3" autoresize class="w-full" placeholder="mis. Query memakai placeholder ? sehingga aman dari SQL injection" />
        </UFormField>

        <UFormField
          v-if="verdict === 'false_positive'"
          label="Saran pembaruan rule audit (opsional)"
          description="Bagaimana rule/prompt audit sebaiknya diubah agar tidak keliru lagi."
        >
          <UTextarea v-model="ruleAdvice" :rows="2" autoresize class="w-full" placeholder="mis. Abaikan db.Raw bila argumen kedua dst. dipakai sebagai parameter terikat" />
        </UFormField>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton color="neutral" variant="ghost" @click="open = false">
          Batal
        </UButton>
        <UButton color="primary" icon="i-lucide-save" :loading="saving" @click="save">
          Simpan vonis
        </UButton>
      </div>
    </template>
  </UModal>
</template>
