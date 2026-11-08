<script setup lang="ts">
import { renderMarkdown } from '~/utils/markdownSanitizer'

const props = withDefaults(
  defineProps<{
    content?: string | null
    compact?: boolean
    maxLines?: number
    emptyText?: string
  }>(),
  {
    content: '',
    compact: false,
    maxLines: undefined,
    emptyText: 'Tidak ada keterangan.'
  }
)

const renderedHtml = computed(() => renderMarkdown(props.content))

const clampStyle = computed(() => {
  if (!props.maxLines) return undefined
  return {
    display: '-webkit-box',
    WebkitLineClamp: props.maxLines,
    WebkitBoxOrient: 'vertical',
    overflow: 'hidden'
  } as Record<string, string | number>
})
</script>

<template>
  <div v-if="!content?.trim()" class="text-xs text-neutral-400 italic font-mono py-1">
    {{ emptyText }}
  </div>

  <div
    v-else
    class="markdown-viewer"
    :class="[
      compact ? 'markdown-compact text-xs' : 'markdown-full text-xs sm:text-sm'
    ]"
    :style="clampStyle"
    v-html="renderedHtml"
  />
</template>

<style scoped>
/* Base typography */
.markdown-viewer {
  word-break: break-word;
}

/* Compact mode (for list cards) */
.markdown-compact :deep(p) {
  margin: 0.25rem 0;
  line-height: 1.5;
  color: inherit;
}
.markdown-compact :deep(p:first-child) {
  margin-top: 0;
}
.markdown-compact :deep(p:last-child) {
  margin-bottom: 0;
}
.markdown-compact :deep(strong),
.markdown-compact :deep(b) {
  font-weight: 600;
  color: var(--color-neutral-900, #171717);
}
:root.dark .markdown-compact :deep(strong),
:root.dark .markdown-compact :deep(b) {
  color: var(--color-neutral-100, #f5f5f5);
}
.markdown-compact :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.72rem;
  padding: 0.1rem 0.35rem;
  border-radius: 0.25rem;
  background-color: var(--color-neutral-100, #f5f5f5);
  color: var(--color-sky-700, #0369a1);
  border: 1px solid rgba(0, 0, 0, 0.08);
}
:root.dark .markdown-compact :deep(code) {
  background-color: var(--color-neutral-800, #262626);
  color: var(--color-sky-400, #38bdf8);
  border-color: rgba(255, 255, 255, 0.1);
}
.markdown-compact :deep(ul),
.markdown-compact :deep(ol) {
  margin: 0.25rem 0;
  padding-left: 1.1rem;
}
.markdown-compact :deep(li) {
  margin: 0.1rem 0;
  list-style-type: disc;
}
.markdown-compact :deep(h1),
.markdown-compact :deep(h2),
.markdown-compact :deep(h3),
.markdown-compact :deep(h4) {
  font-size: 0.8rem;
  font-weight: 600;
  margin: 0.25rem 0 0.15rem 0;
  color: var(--color-neutral-900, #171717);
}
:root.dark .markdown-compact :deep(h1),
:root.dark .markdown-compact :deep(h2),
:root.dark .markdown-compact :deep(h3),
:root.dark .markdown-compact :deep(h4) {
  color: var(--color-white, #ffffff);
}
.markdown-compact :deep(pre) {
  background-color: #0d1117;
  color: #c9d1d9;
  padding: 0.45rem 0.65rem;
  border-radius: 0.375rem;
  overflow-x: auto;
  font-size: 0.72rem;
  margin: 0.35rem 0;
  border: 1px solid #30363d;
}
.markdown-compact :deep(pre code) {
  background: transparent;
  padding: 0;
  border: none;
  color: inherit;
  font-size: inherit;
}
.markdown-compact :deep(blockquote) {
  border-left: 2px solid var(--color-sky-500, #0ea5e9);
  padding-left: 0.5rem;
  margin: 0.35rem 0;
  font-style: italic;
  color: var(--color-neutral-600, #525252);
  background-color: var(--color-neutral-50, #fafafa);
  border-radius: 0 0.25rem 0.25rem 0;
  padding-top: 0.15rem;
  padding-bottom: 0.15rem;
}
:root.dark .markdown-compact :deep(blockquote) {
  color: var(--color-neutral-400, #a3a3a3);
  background-color: rgba(38, 38, 38, 0.4);
}
.markdown-compact :deep(a) {
  color: var(--color-sky-600, #0284c7);
  text-decoration: underline;
  text-underline-offset: 2px;
}
:root.dark .markdown-compact :deep(a) {
  color: var(--color-sky-400, #38bdf8);
}
.markdown-compact :deep(hr) {
  border: 0;
  border-top: 1px solid var(--color-neutral-200, #e5e5e5);
  margin: 0.5rem 0;
}
:root.dark .markdown-compact :deep(hr) {
  border-top-color: var(--color-neutral-800, #262626);
}
.markdown-compact :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0.35rem 0;
  font-size: 0.72rem;
}
.markdown-compact :deep(th),
.markdown-compact :deep(td) {
  padding: 0.25rem 0.4rem;
  border: 1px solid var(--color-neutral-200, #e5e5e5);
}
:root.dark .markdown-compact :deep(th),
:root.dark .markdown-compact :deep(td) {
  border-color: var(--color-neutral-800, #262626);
}
.markdown-compact :deep(th) {
  background-color: var(--color-neutral-100, #f5f5f5);
  font-weight: 600;
}
:root.dark .markdown-compact :deep(th) {
  background-color: var(--color-neutral-800, #262626);
}

/* Full mode (for detail view) */
.markdown-full {
  line-height: 1.65;
  color: var(--color-neutral-700, #404040);
}
:root.dark .markdown-full {
  color: var(--color-neutral-300, #d4d4d4);
}
.markdown-full :deep(p) {
  margin: 0.5rem 0;
}
.markdown-full :deep(p:first-child) {
  margin-top: 0;
}
.markdown-full :deep(p:last-child) {
  margin-bottom: 0;
}
.markdown-full :deep(h1) {
  font-size: 1.25rem;
  font-weight: 700;
  margin: 1.25rem 0 0.5rem;
  color: var(--color-neutral-900, #171717);
  border-bottom: 1px solid var(--color-neutral-200, #e5e5e5);
  padding-bottom: 0.25rem;
}
:root.dark .markdown-full :deep(h1) {
  color: var(--color-white, #ffffff);
  border-color: var(--color-neutral-800, #262626);
}
.markdown-full :deep(h2) {
  font-size: 1.1rem;
  font-weight: 700;
  margin: 1rem 0 0.4rem;
  color: var(--color-neutral-900, #171717);
}
:root.dark .markdown-full :deep(h2) {
  color: var(--color-white, #ffffff);
}
.markdown-full :deep(h3) {
  font-size: 0.95rem;
  font-weight: 600;
  margin: 0.85rem 0 0.35rem;
  color: var(--color-sky-700, #0369a1);
}
:root.dark .markdown-full :deep(h3) {
  color: var(--color-sky-400, #38bdf8);
}
.markdown-full :deep(h4) {
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0.75rem 0 0.25rem;
  color: var(--color-neutral-800, #262626);
}
:root.dark .markdown-full :deep(h4) {
  color: var(--color-neutral-200, #e5e5e5);
}
.markdown-full :deep(strong),
.markdown-full :deep(b) {
  font-weight: 600;
  color: var(--color-neutral-900, #171717);
}
:root.dark .markdown-full :deep(strong),
:root.dark .markdown-full :deep(b) {
  color: var(--color-white, #ffffff);
}
.markdown-full :deep(ul) {
  list-style-type: disc;
  margin: 0.5rem 0;
  padding-left: 1.25rem;
}
.markdown-full :deep(ol) {
  list-style-type: decimal;
  margin: 0.5rem 0;
  padding-left: 1.25rem;
}
.markdown-full :deep(li) {
  margin: 0.25rem 0;
}
.markdown-full :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.78rem;
  padding: 0.15rem 0.4rem;
  border-radius: 0.3rem;
  background-color: var(--color-neutral-100, #f5f5f5);
  color: var(--color-sky-700, #0369a1);
  border: 1px solid rgba(0, 0, 0, 0.08);
}
:root.dark .markdown-full :deep(code) {
  background-color: var(--color-neutral-800, #262626);
  color: var(--color-sky-400, #38bdf8);
  border-color: rgba(255, 255, 255, 0.1);
}
.markdown-full :deep(pre) {
  background-color: #0d1117;
  color: #c9d1d9;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  overflow-x: auto;
  font-size: 0.75rem;
  margin: 0.75rem 0;
  border: 1px solid #30363d;
}
.markdown-full :deep(pre code) {
  background: transparent;
  padding: 0;
  border: none;
  color: inherit;
  font-size: inherit;
}
.markdown-full :deep(blockquote) {
  border-left: 3px solid var(--color-sky-500, #0ea5e9);
  padding-left: 0.75rem;
  margin: 0.5rem 0;
  font-style: italic;
  color: var(--color-neutral-600, #525252);
  background-color: var(--color-neutral-50, #fafafa);
  border-radius: 0 0.375rem 0.375rem 0;
  padding-top: 0.25rem;
  padding-bottom: 0.25rem;
}
:root.dark .markdown-full :deep(blockquote) {
  color: var(--color-neutral-400, #a3a3a3);
  background-color: rgba(38, 38, 38, 0.4);
}
.markdown-full :deep(a) {
  color: var(--color-sky-600, #0284c7);
  text-decoration: underline;
  text-underline-offset: 2px;
}
:root.dark .markdown-full :deep(a) {
  color: var(--color-sky-400, #38bdf8);
}
.markdown-full :deep(hr) {
  border: 0;
  border-top: 1px solid var(--color-neutral-200, #e5e5e5);
  margin: 1rem 0;
}
:root.dark .markdown-full :deep(hr) {
  border-top-color: var(--color-neutral-800, #262626);
}
</style>
