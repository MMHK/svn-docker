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
      class="fixed inset-0 z-40 flex items-start justify-center overflow-y-auto bg-[#1f2328]/50 p-4"
      @click.self="close"
    >
      <div
        class="mt-8 w-full max-w-lg rounded-lg border border-[#d0d7de] bg-[#ffffff] shadow-xl dark:border-[#30363d] dark:bg-[#161b22]"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
      >
        <div class="flex items-start justify-between gap-3 border-b border-[#d0d7de] px-5 py-3 dark:border-[#30363d]">
          <h2 class="text-base font-semibold">{{ title }}</h2>
          <button
            type="button"
            class="rounded-md px-1 text-xl leading-none text-[#656d76] hover:text-[#1f2328] dark:text-[#8b949e] dark:hover:text-[#e6edf3]"
            :aria-label="t('common.close')"
            @click="close"
          >
            ×
          </button>
        </div>

        <div class="px-5 py-4">
          <slot />
        </div>

        <div v-if="$slots.footer" class="border-t border-[#d0d7de] px-5 py-3 dark:border-[#30363d]">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Teleport>
</template>
