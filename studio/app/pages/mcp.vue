<script setup lang="ts">
const { callMcp, getMCPInfo, apiBase } = useApi()

interface ToolItem {
  name: string
  description: string
  inputSchema?: {
    type?: string
    properties?: Record<string, any>
    required?: string[]
  }
  defaultArgs?: string
}

const selectedTool = ref('list_projects')
const toolArgs = ref('{}')
const executing = ref(false)
const rpcResponse = ref<any>(null)
const executionTimeMs = ref<number | null>(null)
const activeConfigTab = ref<'antigravity' | 'claude' | 'cursor' | 'vscode' | 'http'>('antigravity')

const mcpInfo = ref<any>(null)
const loadingInfo = ref(true)
const availableTools = ref<ToolItem[]>([])
const copiedNotice = ref('')

const defaultToolArgsMap: Record<string, string> = {
  list_projects: '{}',
  get_project_detail: '{\n  "project_id": 1\n}',
  scan_directory: '{\n  "path": "."\n}',
  run_diagnostics: '{\n  "project_id": 1\n}',
  get_ai_recommendations: '{\n  "project_id": 1\n}',
  vibeasik_tutorials: '{\n  "category": "dasar-pemrograman"\n}',
  vibeasik_search_domain: '{\n  "query": "vibecoder"\n}',
  vibeasik_deploy_plan: '{\n  "project_name": "my-app",\n  "language": "go",\n  "framework": "fiber",\n  "port": 8080\n}',
  vibeasik_list_local_agents: '{}',
  vibeasik_run_local_agent: '{\n  "agent_key": "agy",\n  "prompt": "Jelaskan topologi arsitektur proyek ini"\n}',
  vibeasik_ai_templates: '{\n  "type": "system-one"\n}',
  vibeasik_inject_ai_template: '{\n  "template_id": "tmpl-system-one-go",\n  "target_path": "."\n}',
  vibeasik_check_ports: '{\n  "ports": [3000, 8765, 8770]\n}',
  vibeasik_rea_status: '{}',
  vibeasik_rea_analyze: '{\n  "target_path": "/path/to/app",\n  "analysis_type": "javascript"\n}'
}

const loadMCPData = async () => {
  loadingInfo.value = true
  try {
    const info = await getMCPInfo()
    mcpInfo.value = info

    if (info?.tools && Array.isArray(info.tools) && info.tools.length > 0) {
      availableTools.value = info.tools.map((t: any) => ({
        name: t.name,
        description: t.description,
        inputSchema: t.inputSchema,
        defaultArgs: defaultToolArgsMap[t.name] || generateDefaultArgs(t.inputSchema)
      }))
    } else {
      // Fallback via tools/list
      const listRes = await callMcp('tools/list')
      if (listRes?.result?.tools) {
        availableTools.value = listRes.result.tools.map((t: any) => ({
          name: t.name,
          description: t.description,
          inputSchema: t.inputSchema,
          defaultArgs: defaultToolArgsMap[t.name] || generateDefaultArgs(t.inputSchema)
        }))
      }
    }
  } catch (err) {
    console.error('Failed to load MCP info:', err)
  } finally {
    loadingInfo.value = false
  }
}

const generateDefaultArgs = (schema?: any): string => {
  if (!schema?.properties) return '{}'
  const obj: Record<string, any> = {}
  for (const [key, val] of Object.entries<any>(schema.properties)) {
    if (val.type === 'integer' || val.type === 'number') obj[key] = 1
    else if (val.type === 'array') obj[key] = []
    else if (val.type === 'boolean') obj[key] = true
    else obj[key] = ''
  }
  return JSON.stringify(obj, null, 2)
}

onMounted(() => {
  loadMCPData()
})

const onSelectTool = (tool: ToolItem) => {
  selectedTool.value = tool.name
  toolArgs.value = tool.defaultArgs || '{}'
}

const executeMcpTool = async () => {
  executing.value = true
  const start = performance.now()
  try {
    let parsedArgs = {}
    if (toolArgs.value.trim()) {
      parsedArgs = JSON.parse(toolArgs.value)
    }

    const res = await callMcp('tools/call', {
      name: selectedTool.value,
      arguments: parsedArgs
    })
    rpcResponse.value = res
  } catch (err: any) {
    rpcResponse.value = {
      error: err.data || err.message || 'Error executing MCP call'
    }
  } finally {
    executionTimeMs.value = Math.round(performance.now() - start)
    executing.value = false
  }
}

const copyText = (txt: string, label = 'Tersalin!') => {
  navigator.clipboard.writeText(txt)
  copiedNotice.value = label
  setTimeout(() => {
    copiedNotice.value = ''
  }, 2000)
}

const cmdExecutable = computed(() => {
  if (mcpInfo.value?.in_path) return 'vibeasik'
  return mcpInfo.value?.binary_path || 'vibeasik'
})

const antigravitySnippet = computed(() => {
  return mcpInfo.value?.snippets?.agy_command || `agy mcp add vibeasik -- ${cmdExecutable.value} mcp`
})

const claudeConfigSnippet = computed(() => {
  const cfg = mcpInfo.value?.snippets?.claude || {
    mcpServers: {
      vibeasik: {
        command: cmdExecutable.value,
        args: ['mcp']
      }
    }
  }
  return JSON.stringify(cfg, null, 2)
})

const cursorConfigSnippet = computed(() => {
  const cfg = mcpInfo.value?.snippets?.cursor || {
    mcpServers: {
      vibeasik: {
        command: cmdExecutable.value,
        args: ['mcp']
      }
    }
  }
  return JSON.stringify(cfg, null, 2)
})

const vscodeConfigSnippet = computed(() => {
  return JSON.stringify({
    mcpServers: {
      vibeasik: {
        command: cmdExecutable.value,
        args: ['mcp']
      }
    }
  }, null, 2)
})

const httpConfigSnippet = computed(() => {
  const url = mcpInfo.value?.http_url || `${apiBase}/mcp`
  return JSON.stringify({
    mcpServers: {
      vibeasik: {
        url
      }
    }
  }, null, 2)
})

const sseConfigSnippet = computed(() => {
  const url = mcpInfo.value?.sse_url || `${apiBase}/mcp/sse`
  return JSON.stringify({
    mcpServers: {
      vibeasikSSE: {
        url
      }
    }
  }, null, 2)
})
</script>

<template>
  <div class="space-y-8 max-w-5xl mx-auto">
    <!-- Header & System Status -->
    <div class="pb-4 border-b border-neutral-200 dark:border-white/10 space-y-3">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-sky-600 dark:bg-sky-500 flex items-center justify-center text-white shadow-xs">
            <UIcon name="i-lucide-bot" class="w-6 h-6" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h1 class="text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
                Model Context Protocol (MCP) Hub
              </h1>
              <UBadge color="primary" variant="subtle" size="sm">
                {{ availableTools.length || 13 }} Tools
              </UBadge>
            </div>
            <p class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5">
              Hubungkan agen AI eksternal (Google Antigravity, Claude Code, Cursor, VS Code Cline) ke workspace proyek lokal Anda secara aman.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <NuxtLink to="/ai-hub">
            <UButton icon="i-lucide-sparkles" color="primary" variant="subtle" size="sm">
              Terminal Agent Hub
            </UButton>
          </NuxtLink>
        </div>
      </div>

      <!-- Quick 1-Click Auto Install Banner for Users who downloaded Vibeasik -->
      <div class="p-4 rounded-xl border border-sky-200 dark:border-sky-900/60 bg-sky-50/50 dark:bg-sky-950/20 flex flex-col md:flex-row items-start md:items-center justify-between gap-3">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-zap" class="w-4 h-4 text-sky-600 dark:text-sky-400" />
            <span class="text-xs font-bold text-sky-900 dark:text-sky-200">
              Auto-Register MCP untuk User vibeasik
            </span>
            <UBadge size="xs" :color="mcpInfo?.in_path ? 'success' : 'neutral'" variant="subtle">
              {{ mcpInfo?.in_path ? 'Terpasang di PATH' : 'CLI Tersedia' }}
            </UBadge>
          </div>
          <p class="text-xs text-sky-800 dark:text-sky-300">
            Jalankan perintah ini di terminal laptop Anda untuk mendaftarkan vibeasik ke Claude Desktop, Cursor, dan Antigravity otomatis:
          </p>
        </div>

        <div class="flex items-center gap-2 w-full md:w-auto">
          <code class="px-3 py-1.5 rounded-lg bg-[#010810] text-sky-300 border border-white/10 font-mono text-xs select-all shrink-0">
            vibeasik mcp install
          </code>
          <UButton
            icon="i-lucide-copy"
            color="primary"
            variant="solid"
            size="xs"
            @click="copyText('vibeasik mcp install', 'Perintah install disalin!')"
          >
            Salin
          </UButton>
        </div>
      </div>
    </div>

    <!-- Multi-Agent Config Cards -->
    <div class="p-6 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-neutral-200 dark:border-white/10 pb-3">
        <div>
          <h2 class="text-base font-bold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-settings-2" class="w-4 h-4 text-sky-500" />
            Pilih Konfigurasi AI Agent Anda
          </h2>
          <p class="text-xs text-neutral-500 dark:text-slate-400">
            Salin konfigurasi spesifik sesuai aplikasi AI yang Anda gunakan di laptop.
          </p>
        </div>

        <!-- Navigation Tabs -->
        <div class="flex items-center bg-neutral-100 dark:bg-slate-800/80 p-1 rounded-xl gap-1 shrink-0 overflow-x-auto max-w-full border border-neutral-200/60 dark:border-white/10">
          <button
            type="button"
            class="px-3 py-1 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5 whitespace-nowrap"
            :class="activeConfigTab === 'antigravity' ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-xs font-semibold' : 'text-neutral-500 dark:text-slate-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeConfigTab = 'antigravity'"
          >
            <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5 text-sky-500" />
            Antigravity
          </button>
          <button
            type="button"
            class="px-3 py-1 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5 whitespace-nowrap"
            :class="activeConfigTab === 'claude' ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-xs font-semibold' : 'text-neutral-500 dark:text-slate-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeConfigTab = 'claude'"
          >
            <UIcon name="i-lucide-bot" class="w-3.5 h-3.5 text-sky-500" />
            Claude Desktop / Code
          </button>
          <button
            type="button"
            class="px-3 py-1 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5 whitespace-nowrap"
            :class="activeConfigTab === 'cursor' ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-xs font-semibold' : 'text-neutral-500 dark:text-slate-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeConfigTab = 'cursor'"
          >
            <UIcon name="i-lucide-terminal" class="w-3.5 h-3.5 text-sky-500" />
            Cursor
          </button>
          <button
            type="button"
            class="px-3 py-1 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5 whitespace-nowrap"
            :class="activeConfigTab === 'vscode' ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-xs font-semibold' : 'text-neutral-500 dark:text-slate-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeConfigTab = 'vscode'"
          >
            <UIcon name="i-lucide-code-2" class="w-3.5 h-3.5 text-sky-500" />
            VS Code / Cline
          </button>
          <button
            type="button"
            class="px-3 py-1 rounded-lg text-xs font-medium transition-all cursor-pointer flex items-center gap-1.5 whitespace-nowrap"
            :class="activeConfigTab === 'http' ? 'bg-white dark:bg-slate-900 text-sky-600 dark:text-sky-400 shadow-xs font-semibold' : 'text-neutral-500 dark:text-slate-400 hover:text-neutral-900 dark:hover:text-white'"
            @click="activeConfigTab = 'http'"
          >
            <UIcon name="i-lucide-radio" class="w-3.5 h-3.5 text-sky-500" />
            HTTP & SSE
          </button>
        </div>
      </div>

      <!-- Tab Content: Antigravity -->
      <div v-if="activeConfigTab === 'antigravity'" class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-xs text-neutral-600 dark:text-neutral-400">
            Jalankan perintah ini di terminal laptop untuk mendaftarkan vibeasik ke Google Antigravity (agy):
          </p>
          <UButton icon="i-lucide-copy" color="neutral" variant="ghost" size="xs" @click="copyText(antigravitySnippet, 'Perintah agy disalin!')">
            Salin CLI Command
          </UButton>
        </div>
        <pre class="p-3.5 rounded-xl bg-[#010810] border border-white/10 text-sky-300 font-mono text-xs overflow-x-auto">{{ antigravitySnippet }}</pre>
        <p class="text-[11px] text-neutral-400">
          Atau gunakan berkas <code class="text-xs">~/.gemini/antigravity/mcp_config.json</code> dengan format stdio standar.
        </p>
      </div>

      <!-- Tab Content: Claude -->
      <div v-else-if="activeConfigTab === 'claude'" class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-xs text-neutral-600 dark:text-neutral-400">
            Tambahkan ke file <code class="font-mono text-xs text-sky-600 dark:text-sky-400">claude_desktop_config.json</code> atau jalankan via Claude Code:
          </p>
          <div class="flex items-center gap-2">
            <UButton icon="i-lucide-terminal" color="neutral" variant="ghost" size="xs" @click="copyText(`claude mcp add vibeasik -- ${cmdExecutable} mcp`, 'Perintah Claude Code disalin!')">
              Salin CLI
            </UButton>
            <UButton icon="i-lucide-copy" color="neutral" variant="ghost" size="xs" @click="copyText(claudeConfigSnippet, 'JSON disalin!')">
              Salin JSON
            </UButton>
          </div>
        </div>
        <pre class="p-3.5 rounded-xl bg-[#010810] border border-white/10 text-sky-300 font-mono text-xs overflow-x-auto">{{ claudeConfigSnippet }}</pre>
      </div>

      <!-- Tab Content: Cursor -->
      <div v-else-if="activeConfigTab === 'cursor'" class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-xs text-neutral-600 dark:text-neutral-400">
            Simpan di berkas konfigurasi Cursor: <code class="font-mono text-xs text-sky-600 dark:text-sky-400">~/.cursor/mcp.json</code>:
          </p>
          <UButton icon="i-lucide-copy" color="neutral" variant="ghost" size="xs" @click="copyText(cursorConfigSnippet, 'JSON Cursor disalin!')">
            Salin JSON
          </UButton>
        </div>
        <pre class="p-3.5 rounded-xl bg-[#010810] border border-white/10 text-sky-300 font-mono text-xs overflow-x-auto">{{ cursorConfigSnippet }}</pre>
      </div>

      <!-- Tab Content: VS Code / Cline -->
      <div v-else-if="activeConfigTab === 'vscode'" class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-xs text-neutral-600 dark:text-neutral-400">
            Konfigurasi untuk ekstensi Cline / Roo Code / Continue di VS Code:
          </p>
          <UButton icon="i-lucide-copy" color="neutral" variant="ghost" size="xs" @click="copyText(vscodeConfigSnippet, 'JSON VS Code disalin!')">
            Salin JSON
          </UButton>
        </div>
        <pre class="p-3.5 rounded-xl bg-[#010810] border border-white/10 text-sky-300 font-mono text-xs overflow-x-auto">{{ vscodeConfigSnippet }}</pre>
      </div>

      <!-- Tab Content: HTTP & SSE -->
      <div v-else class="space-y-3">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div class="p-3 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50/50 dark:bg-slate-800/30 space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-neutral-900 dark:text-white">HTTP JSON-RPC Transport</span>
              <UButton icon="i-lucide-copy" color="neutral" variant="ghost" size="xs" @click="copyText(httpConfigSnippet)">
                Salin
              </UButton>
            </div>
            <pre class="p-2.5 rounded-md bg-[#010810] border border-white/10 text-sky-300 font-mono text-xs overflow-x-auto">{{ httpConfigSnippet }}</pre>
          </div>

          <div class="p-3 rounded-lg border border-neutral-200 dark:border-white/10 bg-neutral-50/50 dark:bg-slate-800/30 space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-neutral-900 dark:text-white">Server-Sent Events (SSE) Transport</span>
              <UButton icon="i-lucide-copy" color="neutral" variant="ghost" size="xs" @click="copyText(sseConfigSnippet)">
                Salin
              </UButton>
            </div>
            <pre class="p-2.5 rounded-md bg-[#010810] border border-white/10 text-sky-300 font-mono text-xs overflow-x-auto">{{ sseConfigSnippet }}</pre>
          </div>
        </div>
      </div>

      <!-- Copied Alert Badge -->
      <div v-if="copiedNotice" class="text-xs text-sky-600 dark:text-sky-400 font-semibold flex items-center gap-1.5 pt-1">
        <UIcon name="i-lucide-check-circle" class="w-4 h-4" />
        {{ copiedNotice }}
      </div>
    </div>

    <!-- Live MCP Tool Runner Sandbox -->
    <div class="p-6 rounded-2xl border border-neutral-200 dark:border-white/10 bg-white dark:bg-slate-900/40 shadow-xs space-y-5">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-lg font-bold text-neutral-900 dark:text-white flex items-center gap-2">
            <UIcon name="i-lucide-play-circle" class="w-5 h-5 text-sky-500" />
            Interactive MCP Tool Sandbox
          </h2>
          <p class="text-xs text-neutral-500 mt-0.5">
            Uji coba pemanggilan alat MCP langsung via JSON-RPC 2.0 untuk melihat konteks terstruktur yang diterima AI agent.
          </p>
        </div>

        <UBadge color="neutral" variant="subtle" size="sm" class="font-mono">
          Protocol: 2024-11-05
        </UBadge>
      </div>

      <!-- Tool Selector Grid -->
      <div class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-semibold uppercase tracking-wider text-neutral-500">
            Pilih Tool MCP ({{ availableTools.length }})
          </span>
          <span class="text-[11px] text-neutral-400 font-mono">
            Aktif: <strong class="text-sky-600 dark:text-sky-400">{{ selectedTool }}</strong>
          </span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5 max-h-72 overflow-y-auto pr-1">
          <button
            v-for="t in availableTools"
            :key="t.name"
            class="p-3 rounded-xl border text-left transition-all cursor-pointer space-y-1.5"
            :class="selectedTool === t.name
              ? 'border-sky-500 bg-sky-50/50 dark:bg-sky-950/30 text-sky-900 dark:text-sky-100 shadow-xs ring-1 ring-sky-500'
              : 'border-neutral-200 dark:border-white/10 hover:border-sky-500/50 bg-neutral-50/50 dark:bg-slate-800/30 text-neutral-700 dark:text-slate-300'"
            @click="onSelectTool(t)"
          >
            <div class="flex items-center justify-between gap-1">
              <span class="font-mono font-bold text-xs truncate">{{ t.name }}</span>
              <UIcon
                v-if="selectedTool === t.name"
                name="i-lucide-check"
                class="w-3.5 h-3.5 text-sky-600 dark:text-sky-400 shrink-0"
              />
            </div>
            <div class="text-[11px] text-neutral-500 dark:text-neutral-400 line-clamp-2 leading-relaxed">
              {{ t.description }}
            </div>
          </button>
        </div>
      </div>

      <!-- Arguments Editor -->
      <div class="space-y-2">
        <div class="flex items-center justify-between text-xs">
          <label class="font-semibold uppercase text-neutral-500">Argumen Tool (JSON Payload)</label>
          <span class="font-mono text-neutral-400 text-[11px]">params.arguments</span>
        </div>
        <textarea
          v-model="toolArgs"
          rows="4"
          class="w-full font-mono text-xs p-3.5 rounded-xl border border-neutral-200 dark:border-neutral-800 bg-neutral-50 dark:bg-[#010810] text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-sky-500"
        />
      </div>

      <!-- Action Button -->
      <div class="flex items-center justify-between">
        <UButton
          icon="i-lucide-send"
          color="primary"
          size="md"
          :loading="executing"
          @click="executeMcpTool"
        >
          Eksekusi Tool MCP
        </UButton>

        <span v-if="executionTimeMs !== null" class="text-xs font-mono text-neutral-400">
          Latency: {{ executionTimeMs }}ms
        </span>
      </div>

      <!-- Response Box -->
      <div v-if="rpcResponse" class="space-y-2 pt-3 border-t border-neutral-100 dark:border-neutral-800">
        <div class="flex items-center justify-between text-xs">
          <span class="font-semibold uppercase text-neutral-500">Respon MCP JSON-RPC</span>
          <button
            class="text-sky-600 dark:text-sky-400 hover:underline font-mono text-xs cursor-pointer flex items-center gap-1"
            @click="copyText(JSON.stringify(rpcResponse, null, 2), 'Respon berhasil disalin!')"
          >
            <UIcon name="i-lucide-copy" class="w-3 h-3" />
            Salin Respon
          </button>
        </div>
        <pre class="p-4 rounded-xl bg-[#010810] border border-white/10 text-sky-300 font-mono text-xs overflow-x-auto max-h-96 whitespace-pre-wrap leading-relaxed">{{ JSON.stringify(rpcResponse, null, 2) }}</pre>
      </div>
    </div>
  </div>
</template>
