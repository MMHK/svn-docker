<script setup lang="ts">
import { t } from '../i18n';
import { useToastStore } from '../stores/toast';

const toasts = useToastStore();
</script>

<template>
  <div
    class="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-center gap-2 px-4 sm:items-end sm:pr-6"
    aria-live="polite"
  >
    <div
      v-for="toast in toasts.items"
      :key="toast.id"
      class="pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-md border px-4 py-3 text-sm shadow-lg"
      :class="
        toast.tone === 'error'
          ? 'border-red-300 bg-red-50 text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200'
          : 'border-emerald-300 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-200'
      "
      role="status"
    >
      <p class="flex-1 break-words">{{ toast.message }}</p>
      <button
        type="button"
        class="shrink-0 rounded-md px-1 text-base leading-none opacity-70 hover:opacity-100"
        :aria-label="t('common.close')"
        @click="toasts.dismiss(toast.id)"
      >
        ×
      </button>
    </div>
  </div>
</template>
