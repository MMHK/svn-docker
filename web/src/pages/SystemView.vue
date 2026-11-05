<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { getSystemInfo } from '../api/client';
import { toError } from '../api/errors';
import type { PathInfo, SystemInfo } from '../api/types';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';

const info = ref<SystemInfo | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);

const paths = computed<PathInfo[]>(() => {
  const value = info.value;

  if (value === null) {
    return [];
  }

  return [value.svnParentPath, value.svnAuthFile, value.svnAccessFile];
});

function flag(value: boolean): string {
  return value ? t('common.yes') : t('common.no');
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    info.value = await getSystemInfo();
  } catch (cause) {
    info.value = null;
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
  <section class="space-y-6">
    <div class="flex flex-wrap items-center gap-3">
      <h1 class="text-xl font-semibold">{{ t('system.title') }}</h1>
      <button
        type="button"
        class="rounded-md border border-[#d0d7de] px-3 py-1 text-sm hover:bg-[#f6f8fa] disabled:opacity-50 dark:border-[#30363d] dark:hover:bg-[#21262d]"
        :disabled="loading"
        @click="load"
      >
        {{ t('system.refresh') }}
      </button>
    </div>

    <StateBlock :loading="loading" :error="error" :empty="info === null" :empty-text="t('system.empty')">
      <template v-if="info">
        <dl class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div class="rounded-md border border-[#d0d7de] bg-[#ffffff] p-4 dark:border-[#30363d] dark:bg-[#161b22]">
            <dt class="text-xs uppercase tracking-wide text-[#656d76] dark:text-[#8b949e]">{{ t('system.version') }}</dt>
            <dd class="mt-1 font-mono text-sm">{{ info.version }}</dd>
          </div>
          <div class="rounded-md border border-[#d0d7de] bg-[#ffffff] p-4 dark:border-[#30363d] dark:bg-[#161b22]">
            <dt class="text-xs uppercase tracking-wide text-[#656d76] dark:text-[#8b949e]">{{ t('system.goVersion') }}</dt>
            <dd class="mt-1 font-mono text-sm">{{ info.goVersion }}</dd>
          </div>
          <div class="rounded-md border border-[#d0d7de] bg-[#ffffff] p-4 dark:border-[#30363d] dark:bg-[#161b22]">
            <dt class="text-xs uppercase tracking-wide text-[#656d76] dark:text-[#8b949e]">{{ t('system.platform') }}</dt>
            <dd class="mt-1 font-mono text-sm">{{ info.platform }}</dd>
          </div>
          <div class="rounded-md border border-[#d0d7de] bg-[#ffffff] p-4 dark:border-[#30363d] dark:bg-[#161b22]">
            <dt class="text-xs uppercase tracking-wide text-[#656d76] dark:text-[#8b949e]">{{ t('system.docsEnabled') }}</dt>
            <dd class="mt-1 font-mono text-sm">{{ flag(info.docsEnabled) }}</dd>
          </div>
          <div class="rounded-md border border-[#d0d7de] bg-[#ffffff] p-4 dark:border-[#30363d] dark:bg-[#161b22]">
            <dt class="text-xs uppercase tracking-wide text-[#656d76] dark:text-[#8b949e]">
              {{ t('system.defaultLocale') }}
            </dt>
            <dd class="mt-1 font-mono text-sm">{{ info.defaultLocale }}</dd>
          </div>
          <div class="rounded-md border border-[#d0d7de] bg-[#ffffff] p-4 dark:border-[#30363d] dark:bg-[#161b22]">
            <dt class="text-xs uppercase tracking-wide text-[#656d76] dark:text-[#8b949e]">{{ t('system.serverTime') }}</dt>
            <dd class="mt-1 font-mono text-sm">{{ info.serverTime }}</dd>
          </div>
        </dl>

        <div class="mt-6 overflow-hidden rounded-md border border-[#d0d7de] bg-[#ffffff] dark:border-[#30363d] dark:bg-[#161b22]">
          <div class="flex flex-wrap items-center justify-between gap-2 border-b border-[#d0d7de] px-4 py-3 dark:border-[#30363d]">
            <h2 class="text-sm font-semibold">{{ t('system.paths') }}</h2>
            <span class="text-xs text-[#656d76] dark:text-[#8b949e]">{{ t('system.readOnlyNote') }}</span>
          </div>
          <table class="w-full text-left text-sm">
            <thead class="bg-[#f6f8fa] text-xs uppercase tracking-wide text-[#656d76] dark:bg-[#161b22] dark:text-[#8b949e]">
              <tr>
                <th class="px-4 py-2 font-medium">{{ t('system.path') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('system.exists') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('system.readable') }}</th>
                <th class="px-4 py-2 font-medium">{{ t('system.writable') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in paths" :key="item.path" class="border-t border-[#eaeef2] dark:border-[#30363d]">
                <td class="px-4 py-2 font-mono text-xs">{{ item.path }}</td>
                <td class="px-4 py-2">{{ flag(item.exists) }}</td>
                <td class="px-4 py-2">{{ flag(item.readable) }}</td>
                <td class="px-4 py-2">{{ flag(item.writable) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </StateBlock>
  </section>
</template>
