<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { listRoles } from '../api/client';
import { toError } from '../api/errors';
import type { Role } from '../api/types';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';

const items = ref<Role[]>([]);
const loading = ref(false);
const error = ref<Error | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    const list = await listRoles();
    items.value = list.items ?? [];
  } catch (cause) {
    items.value = [];
    error.value = toError(cause);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="space-y-4">
    <div class="flex flex-wrap items-center gap-3">
      <h1 class="text-xl font-semibold">{{ t('roles.title') }}</h1>
      <button
        type="button"
        class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
        :disabled="loading"
        @click="load"
      >
        {{ t('common.refresh') }}
      </button>
    </div>

    <p class="text-sm text-slate-600 dark:text-slate-300">{{ t('roles.intro') }}</p>

    <StateBlock
      :loading="loading"
      :error="error"
      :empty="items.length === 0"
      :empty-text="t('roles.empty')"
    >
      <div class="overflow-x-auto rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <table class="w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('roles.name') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('roles.description') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="role in items" :key="role.name" class="border-t border-slate-100 dark:border-slate-800">
              <td class="px-4 py-2 font-medium">{{ role.name }}</td>
              <td class="px-4 py-2 text-slate-600 dark:text-slate-300">{{ role.description }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </StateBlock>
  </section>
</template>
