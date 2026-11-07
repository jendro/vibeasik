<script setup lang="ts">
import type { ArchitecturePattern, ArchitectureEvaluation } from '~/types'

definePageMeta({
  fluid: true
})

const {
  activeProjectId,
  activeProject,
  fetchProjects
} = useActiveProject()

const {
  getScaleUpPatterns,
  evaluateProjectArchitecture
} = useApi()

const selectedScope = ref<'backend' | 'mobile' | 'frontend'>('backend')
const activeTab = ref<'patterns' | 'blueprints' | 'dissection' | 'mvpp-mcp' | 'audit'>('patterns')
const selectedPatternId = ref<string>('layered')
const selectedComponentId = ref<number | null>(null)

const patterns = ref<ArchitecturePattern[]>([])
const evaluations = ref<ArchitectureEvaluation[]>([])
const loading = ref(true)
const evaluating = ref(false)

const loadPatterns = async () => {
  try {
    patterns.value = await getScaleUpPatterns()
    if (patterns.value.length > 0 && !selectedPatternId.value && patterns.value[0]) {
      selectedPatternId.value = patterns.value[0].id
    }
  } catch (err: any) {
    console.error('Failed to load architecture patterns:', err)
  }
}

const runProjectEvaluation = async () => {
  if (!activeProjectId.value) {
    evaluations.value = []
    return
  }

  evaluating.value = true
  try {
    const res = await evaluateProjectArchitecture(Number(activeProjectId.value))
    evaluations.value = res || []
    if (evaluations.value.length > 0 && evaluations.value[0]) {
      selectedComponentId.value = evaluations.value[0].component_id
    }
  } catch (err: any) {
    console.warn('Evaluation fallback:', err)
  } finally {
    evaluating.value = false
  }
}

const handleSelectPattern = (id: string) => {
  selectedPatternId.value = id
}

const handleViewBlueprint = (id: string) => {
  selectedPatternId.value = id
  activeTab.value = 'blueprints'
}

const handleSelectComponent = (id: number) => {
  selectedComponentId.value = id
}

watch(activeProjectId, () => {
  runProjectEvaluation()
})

onMounted(async () => {
  loading.value = true
  await Promise.all([
    fetchProjects(),
    loadPatterns()
  ])
  if (activeProjectId.value) {
    await runProjectEvaluation()
  }
  loading.value = false
})
</script>

<template>
  <div class="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto space-y-6">
    <!-- Header Component with Scope Selector -->
    <ScaleUpHeader
      :active-project="activeProject"
      :active-tab="activeTab"
      :selected-scope="selectedScope"
      :evaluating="evaluating"
      :backend-count="evaluations.length"
      @update:active-tab="activeTab = $event as any"
      @update:selected-scope="selectedScope = $event"
      @re-evaluate="runProjectEvaluation"
    />

    <!-- Loading State -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20 space-y-3">
      <UIcon name="i-lucide-loader-2" class="w-8 h-8 text-sky-500 animate-spin" />
      <p class="text-sm text-neutral-500">
        Memuat analisis pola arsitektur aplikasi...
      </p>
    </div>

    <!-- Non-Backend Roadmap Scope (Mobile / Frontend) -->
    <div v-else-if="selectedScope !== 'backend'">
      <ScopePreparationCard
        :scope="selectedScope"
        @back-to-backend="selectedScope = 'backend'"
      />
    </div>

    <!-- Active Backend Tabs -->
    <template v-else>
      <!-- Tab 1: Pattern Catalog -->
      <div v-if="activeTab === 'patterns'">
        <PatternCatalogTab
          :patterns="patterns"
          :selected-pattern-id="selectedPatternId"
          @select-pattern="handleSelectPattern"
          @view-blueprint="handleViewBlueprint"
        />
      </div>

      <!-- Tab 2: Folder Structure Blueprints -->
      <div v-else-if="activeTab === 'blueprints'">
        <FolderStructureBlueprintTab
          :patterns="patterns"
          :selected-pattern-id="selectedPatternId"
          @update:selected-pattern-id="handleSelectPattern"
        />
      </div>

      <!-- Tab 3: Code Dissection (Router, Controller, Functions) -->
      <div v-else-if="activeTab === 'dissection'">
        <CodeDissectionTab
          :patterns="patterns"
          :selected-pattern-id="selectedPatternId"
        />
      </div>

      <!-- Tab 4: MVPP & MCP Deep Dive -->
      <div v-else-if="activeTab === 'mvpp-mcp'">
        <MvppAndMcpDeepDiveTab />
      </div>

      <!-- Tab 5: Live Project Architecture Audit -->
      <div v-else-if="activeTab === 'audit'">
        <ProjectArchitectureAuditTab
          :active-project="activeProject"
          :evaluations="evaluations"
          :selected-component-id="selectedComponentId"
          :evaluating="evaluating"
          @select-component="handleSelectComponent"
          @refresh="runProjectEvaluation"
        />
      </div>
    </template>
  </div>
</template>
