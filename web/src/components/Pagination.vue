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
  <div class="flex flex-wrap items-center gap-3 text-sm text-[#656d76] dark:text-[#c9d1d9]">
    <button
      type="button"
      class="rounded-md border border-[#d0d7de] px-3 py-1 hover:bg-[#f6f8fa] disabled:opacity-40 dark:border-[#30363d] dark:hover:bg-[#21262d]"
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
      class="rounded-md border border-[#d0d7de] px-3 py-1 hover:bg-[#f6f8fa] disabled:opacity-40 dark:border-[#30363d] dark:hover:bg-[#21262d]"
      :disabled="page >= totalPages"
      @click="go(page + 1)"
    >
      {{ t('common.next') }}
    </button>
  </div>
</template>
