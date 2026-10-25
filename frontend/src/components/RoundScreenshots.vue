<template>
  <div v-if="round.images?.length" class="round-screenshots">
    <figure v-for="image in round.images" :key="image.index" class="round-shot">
      <button
        type="button"
        class="round-shot-toggle"
        :aria-expanded="isExpanded(image)"
        @click="toggleImage(image)"
      >
        {{ isExpanded(image) ? `隐藏截图（${image.index + 1}）` : `显示截图（${image.index + 1}）` }}
      </button>

      <div v-if="isExpanded(image)" class="round-shot-content">
        <div v-if="image.loading" class="round-shot-status" role="status">
          正在读取截图…
        </div>
        <button
          v-else-if="image.data"
          type="button"
          class="round-shot-image"
          :aria-label="`隐藏题目截图 ${image.index + 1}`"
          @click="collapseImage(image)"
        >
          <img
            :src="image.data"
            :alt="`题目截图 ${image.index + 1}`"
          />
        </button>
        <div v-else class="round-shot-status">准备读取截图…</div>
        <figcaption v-if="image.error">{{ image.error }}</figcaption>
      </div>
    </figure>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  round: { type: Object, required: true },
  scopeKey: { type: String, required: true },
})
const emit = defineEmits(['load'])

const expandedImages = ref(new Set())

watch(() => props.scopeKey, () => {
  expandedImages.value = new Set()
})

function isExpanded(image) {
  return expandedImages.value.has(image.index)
}

function toggleImage(image) {
  if (isExpanded(image)) {
    collapseImage(image)
    return
  }

  const next = new Set(expandedImages.value)
  next.add(image.index)
  expandedImages.value = next

  if (!image.data && !image.loading) {
    emit('load', props.round, image)
  }
}

function collapseImage(image) {
  if (!isExpanded(image)) return
  const next = new Set(expandedImages.value)
  next.delete(image.index)
  expandedImages.value = next
}
</script>

<style scoped>
.round-screenshots {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: var(--sp-3);
  margin-bottom: var(--sp-4);
}
.round-shot {
  min-width: 0;
  margin: 0;
  padding: var(--sp-2);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-card);
}
.round-shot-toggle {
  width: 100%;
  min-height: 2.5rem;
  padding: var(--sp-2) var(--sp-3);
  border: 1px dashed var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-input);
  color: var(--text-secondary);
  cursor: pointer;
}
.round-shot-content {
  margin-top: var(--sp-2);
}
.round-shot-image {
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  cursor: pointer;
}
.round-shot-image img {
  display: block;
  width: 100%;
  max-height: 280px;
  object-fit: contain;
  border-radius: var(--radius-sm);
}
.round-shot-status {
  min-height: 5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-3);
  border-radius: var(--radius-sm);
  background: var(--surface-input);
  color: var(--text-muted);
  font-size: var(--text-sm);
}
.round-shot-toggle:focus-visible,
.round-shot-image:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
.round-shot figcaption {
  margin-top: var(--sp-2);
  color: var(--color-error);
  font-size: var(--text-xs);
  overflow-wrap: anywhere;
}
</style>
