<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue';
import { t } from '../i18n';

/**
 * Shared modal shell: teleported to <body>, closes on Escape or a backdrop
 * click, and keeps the dark-mode surface of the rest of the app. Content and
 * footer are slots so each form owns its own validation and buttons.
 */
const props = defineProps<{
  open: boolean;
  title: string;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
}>();

function close(): void {
  emit('close');
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && props.open) {
    close();
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown);
});
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-40 flex items-start justify-center overflow-y-auto bg-slate-900/50 p-4"
      @click.self="close"
    >
      <div
        class="mt-8 w-full max-w-lg rounded-lg border border-slate-200 bg-white shadow-xl dark:border-slate-700 dark:bg-slate-900"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
      >
        <div class="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-3 dark:border-slate-800">
          <h2 class="text-base font-semibold">{{ title }}</h2>
          <button
            type="button"
            class="rounded px-1 text-xl leading-none text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            :aria-label="t('common.close')"
            @click="close"
          >
            ×
          </button>
        </div>

        <div class="px-5 py-4">
          <slot />
        </div>

        <div v-if="$slots.footer" class="border-t border-slate-200 px-5 py-3 dark:border-slate-800">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Teleport>
</template>
