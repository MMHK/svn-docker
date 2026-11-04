<script setup lang="ts">
import { computed } from 'vue';
import { t } from '../i18n';

const props = defineProps<{
  page: number;
  size: number;
  total: number;
}>();

const emit = defineEmits<{
  (event: 'update:page', page: number): void;
}>();

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / Math.max(1, props.size))));

function go(page: number): void {
  if (page < 1 || page > totalPages.value || page === props.page) {
    return;
  }

  emit('update:page', page);
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-3 text-sm text-slate-600 dark:text-slate-300">
    <button
      type="button"
      class="rounded border border-slate-300 px-3 py-1 hover:bg-slate-100 disabled:opacity-40 dark:border-slate-700 dark:hover:bg-slate-800"
      :disabled="page <= 1"
      @click="go(page - 1)"
    >
      {{ t('common.previous') }}
    </button>
    <span>
      {{ t('common.page') }} {{ page }} / {{ totalPages }} · {{ t('common.total') }} {{ total }}
    </span>
    <button
      type="button"
      class="rounded border border-slate-300 px-3 py-1 hover:bg-slate-100 disabled:opacity-40 dark:border-slate-700 dark:hover:bg-slate-800"
      :disabled="page >= totalPages"
      @click="go(page + 1)"
    >
      {{ t('common.next') }}
    </button>
  </div>
</template>
