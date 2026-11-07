import type { Ref } from 'vue'
import type { ProjectSynergyMap } from '~/types'

// Pencarian & filter metode/tabel pada peta sinergi, beserta data turunannya.
export function useSynergyFilters(synergyMap: Ref<ProjectSynergyMap | null>) {
  const searchQuery = ref('')
  const filterMethod = ref<string>('all')
  const filterTable = ref<string>('all')

  const availableTables = computed(() => synergyTables(synergyMap.value))
  const filteredPages = computed(() => filterSynergyPages(synergyMap.value, searchQuery.value, filterMethod.value, filterTable.value))
  const matrixRows = computed(() => buildSynergyMatrix(filteredPages.value))

  const resetFilters = () => {
    searchQuery.value = ''
    filterMethod.value = 'all'
    filterTable.value = 'all'
  }

  return { searchQuery, filterMethod, filterTable, availableTables, filteredPages, matrixRows, resetFilters }
}
