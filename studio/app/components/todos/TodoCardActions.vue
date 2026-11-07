<script setup lang="ts">
import type { AgentRun, Todo } from '~/types'

const props = defineProps<{
  todo: Todo
  agentRun?: AgentRun
}>()

const emit = defineEmits<{
  (e: 'approve', id: number): void
  (e: 'reject', id: number): void
  (e: 'updateStatus', id: number, status: string): void
  (e: 'delete', id: number): void
  (e: 'dispatch', todo: Todo): void
  (e: 'split', id: number): void
}>()

const isGroup = computed(() => props.todo.source === 'todo_group')
const canSplit = computed(() => isGroup.value && !props.agentRun && (props.todo.status === 'pending' || props.todo.status === 'suggested'))

const { formatDateTime } = useFormatters()
</script>

<template>
  <div class="pt-2 border-t border-neutral-100 dark:border-white/10 flex flex-wrap items-center justify-between gap-2">
    <div class="text-[10px] text-neutral-400 font-mono">
      ID #{{ todo.id }} • {{ formatDateTime(todo.created_at) }}
    </div>

    <div class="flex items-center gap-1.5">
      <!-- Approval buttons for suggested AI items -->
      <template v-if="todo.status === 'suggested'">
        <UButton
          size="xs"
          color="success"
          variant="solid"
          icon="i-lucide-check"
          title="Setujui rekomendasi AI ini dan mulai masukkan ke rencana kerja"
          @click="emit('approve', todo.id)"
        >
          Setujui (Approve)
        </UButton>
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-x"
          title="Tolak saran AI"
          @click="emit('reject', todo.id)"
        >
          Tolak
        </UButton>
      </template>

      <!-- Standard workflow status buttons -->
      <template v-else>
        <UButton
          v-if="todo.status !== 'completed' && todo.status !== 'rejected'"
          size="xs"
          color="neutral"
          variant="outline"
          icon="i-lucide-check-circle"
          title="Tandai selesai"
          @click="emit('updateStatus', todo.id, 'completed')"
        >
          Selesai
        </UButton>

        <UButton
          v-if="agentRun"
          size="xs"
          color="info"
          variant="soft"
          icon="i-lucide-loader-circle"
          :ui="{ leadingIcon: agentRun.status === 'running' ? 'animate-spin' : '' }"
          :title="agentRun.status === 'running' ? agentRun.route_reason : (agentRun.wait_reason || 'Menunggu worker yang tersedia')"
          to="/queue"
        >
          <template v-if="agentRun.kind === 'merge'">
            {{ agentRun.status === 'running' ? (agentRun.profile_name || 'Agent') + ' me-merge' : 'Antre merge' }}
          </template>
          <template v-else>
            {{ agentRun.status === 'running' ? (agentRun.profile_name || 'Agent') + ' bekerja' : 'Antre di agent' }}
          </template>
        </UButton>
        <UButton
          v-else-if="todo.status !== 'completed' && todo.status !== 'rejected'"
          size="xs"
          color="primary"
          variant="soft"
          icon="i-lucide-bot"
          title="Kirim todo ini ke agent coding di laptop (Claude Code, Antigravity, ...)"
          @click="emit('dispatch', todo)"
        >
          Suruh Agent
        </UButton>

        <UButton
          v-if="todo.status === 'pending'"
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-play"
          title="Mulai kerjakan"
          @click="emit('updateStatus', todo.id, 'in_progress')"
        >
          Kerjakan
        </UButton>

        <UButton
          v-if="todo.status === 'completed' || todo.status === 'rejected'"
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-rotate-ccw"
          :title="todo.status === 'rejected' ? 'Pulihkan agenda yang ditolak' : 'Buka kembali agenda'"
          @click="emit('updateStatus', todo.id, 'pending')"
        >
          {{ todo.status === 'rejected' ? 'Pulihkan' : 'Buka Lagi' }}
        </UButton>
      </template>

      <UButton
        v-if="canSplit"
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-lucide-split"
        title="Pecah todo gabungan ini kembali menjadi todo biasa"
        @click="emit('split', todo.id)"
      >
        Pecah
      </UButton>

      <!-- Popover Verifikasi Hapus Agenda -->
      <UPopover :content="{ align: 'end', side: 'top' }">
        <UButton
          size="xs"
          color="neutral"
          variant="ghost"
          icon="i-lucide-trash-2"
          class="text-neutral-400 hover:text-red-500 transition-colors"
          title="Hapus agenda"
        />

        <template #content="{ close }">
          <div class="p-3 w-64 space-y-2.5">
            <div class="flex items-start gap-2.5">
              <div class="p-1.5 rounded-lg bg-red-50 dark:bg-red-950/50 text-red-600 dark:text-red-400 shrink-0">
                <UIcon name="i-lucide-alert-triangle" class="w-4 h-4" />
              </div>
              <div class="space-y-0.5">
                <p class="text-xs font-semibold text-neutral-900 dark:text-neutral-100">
                  Hapus Agenda?
                </p>
                <p class="text-[11px] text-neutral-500 dark:text-neutral-400 leading-snug">
                  {{ isGroup ? 'Todo gabungan dihapus; sub todo yang belum selesai kembali menjadi todo biasa.' : 'Agenda ini akan dihapus permanen dari daftar tugas.' }}
                </p>
              </div>
            </div>
            <div class="flex items-center justify-end gap-1.5 pt-2 border-t border-neutral-100 dark:border-white/10">
              <UButton
                size="xs"
                color="neutral"
                variant="ghost"
                label="Batal"
                @click="close"
              />
              <UButton
                size="xs"
                color="error"
                variant="solid"
                icon="i-lucide-trash-2"
                label="Ya, Hapus"
                @click="() => { close?.(); emit('delete', todo.id); }"
              />
            </div>
          </div>
        </template>
      </UPopover>
    </div>
  </div>
</template>
