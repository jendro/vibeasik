<script setup lang="ts">
import type { Todo } from '~/types/todo'
import { parseTodoTags, todoTagMeta } from '~/utils/todoTags'

const props = defineProps<{ todo: Todo }>()
const tags = computed(() => parseTodoTags(props.todo.tags).map(todoTagMeta))
</script>

<template>
  <span
    v-for="tag in tags"
    :key="tag.key"
    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-medium text-xs border"
    :class="tag.class"
  >
    <UIcon :name="tag.icon" class="w-3.5 h-3.5" />
    {{ tag.label }}
  </span>
</template>
