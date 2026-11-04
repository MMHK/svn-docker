<script setup lang="ts">
import { computed } from 'vue';
import { describeError } from '../api/errors';
import { t } from '../i18n';

/**
 * Shared loading / error / empty presenter so no page can render a blank
 * screen (401, 403, 5xx, empty list and in-flight states all have copy).
 */
const props = withDefaults(
  defineProps<{
    loading?: boolean;
    error?: Error | null;
    empty?: boolean;
    emptyText?: string;
  }>(),
  {
    loading: false,
    error: null,
    empty: false,
    emptyText: '',
  },
);

const described = computed(() => (props.error === null ? null : describeError(props.error)));
</script>

<template>
  <p v-if="loading" class="text-sm text-slate-500 dark:text-slate-400" role="status">
    {{ t('common.loading') }}
  </p>

  <div
    v-else-if="described"
    class="rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
    role="alert"
  >
    <p class="font-medium">{{ described.title }}</p>
    <p class="mt-1 break-words">{{ described.detail }}</p>
  </div>

  <p
    v-else-if="empty"
    class="rounded border border-dashed border-slate-300 bg-white px-4 py-6 text-center text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-400"
  >
    {{ emptyText === '' ? t('common.empty') : emptyText }}
  </p>

  <slot v-else />
</template>
