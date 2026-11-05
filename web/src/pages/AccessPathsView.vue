<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { ApiError, createAccessPath, listAccessPaths } from '../api/client';
import { encodeAccessPath } from '../api/encoding';
import { toError } from '../api/errors';
import type { AccessPath } from '../api/types';
import ModalDialog from '../components/ModalDialog.vue';
import Pagination from '../components/Pagination.vue';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';
import { useToastStore } from '../stores/toast';

const toast = useToastStore();

const items = ref<AccessPath[]>([]);
const page = ref(1);
const size = 20;
const total = ref(0);
const query = ref('');
const appliedQuery = ref('');
const loading = ref(false);
const error = ref<Error | null>(null);

/** Mirrors api/pkg/model/accesspath.go (plan section 5.4). */
const PATH_PATTERN = /^[A-Za-z0-9_-]{1,64}:\/.*$/;

const createOpen = ref(false);
const createPath = ref('');
const createError = ref('');
const creating = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    const list = await listAccessPaths(page.value, size, appliedQuery.value);
    items.value = list.items ?? [];
    page.value = list.page;
    total.value = list.total;
  } catch (cause) {
    items.value = [];
    total.value = 0;
    error.value = toError(cause);
  } finally {
    loading.value = false;
  }
}

function search(): void {
  appliedQuery.value = query.value.trim();
  page.value = 1;
  void load();
}

function changePage(next: number): void {
  page.value = next;
  void load();
}

function openCreate(): void {
  createPath.value = '';
  createError.value = '';
  createOpen.value = true;
}

function closeCreate(): void {
  if (creating.value) {
    return;
  }

  createOpen.value = false;
}

/** Rejects the vectors from the plan before the request is sent. */
function validateCreate(): string {
  const path = createPath.value.trim();

  if (path === '') {
    return t('accessPaths.create.pathRequired');
  }

  if (!PATH_PATTERN.test(path) || path.includes('..') || /[\\\u0000-\u001f]/.test(path)) {
    return t('accessPaths.create.pathInvalid');
  }

  return '';
}

function createErrorMessage(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('accessPaths.create.failed');
  }

  switch (cause.status) {
    case 0:
      return t('common.networkErrorHint');
    case 403:
      return t('accessPaths.create.forbidden');
    case 409:
      return t('accessPaths.create.exists');
    case 422:
      return cause.message;
    default:
      return `${t('accessPaths.create.failed')} (${cause.status})`;
  }
}

async function submitCreate(): Promise<void> {
  if (creating.value) {
    return;
  }

  createError.value = validateCreate();
  if (createError.value !== '') {
    return;
  }

  creating.value = true;

  try {
    await createAccessPath(createPath.value.trim());
    createOpen.value = false;
    toast.push(t('accessPaths.create.success'));
    page.value = 1;
    await load();
  } catch (cause) {
    createError.value = createErrorMessage(cause);
  } finally {
    creating.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="space-y-4">
    <div class="flex flex-wrap items-center gap-3">
      <h1 class="text-xl font-semibold">{{ t('accessPaths.title') }}</h1>
      <button
        type="button"
        class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
        @click="openCreate"
      >
        {{ t('accessPaths.create.button') }}
      </button>
      <button
        type="button"
        class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
        :disabled="loading"
        @click="load"
      >
        {{ t('common.refresh') }}
      </button>
    </div>

    <form class="flex flex-wrap gap-2" @submit.prevent="search">
      <input
        v-model="query"
        type="search"
        :placeholder="t('accessPaths.searchPlaceholder')"
        :aria-label="t('common.search')"
        class="w-64 rounded border border-slate-300 bg-white px-3 py-1 text-sm dark:border-slate-700 dark:bg-slate-800"
      />
      <button
        type="submit"
        class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
      >
        {{ t('common.search') }}
      </button>
    </form>

    <StateBlock
      :loading="loading"
      :error="error"
      :empty="items.length === 0"
      :empty-text="t('accessPaths.empty')"
    >
      <div class="overflow-x-auto rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <table class="w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('accessPaths.path') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('accessPaths.acl') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="path in items" :key="path.path" class="border-t border-slate-100 dark:border-slate-800">
              <td class="px-4 py-2">
                <RouterLink
                  :to="{ name: 'access-path-detail', params: { path: encodeAccessPath(path.path) } }"
                  class="font-mono text-xs font-medium text-sky-700 hover:underline dark:text-sky-400"
                >
                  {{ path.path }}
                </RouterLink>
              </td>
              <td class="px-4 py-2">
                <span v-if="path.acl.length === 0" class="text-slate-400 dark:text-slate-500">{{ t('common.none') }}</span>
                <span v-else class="flex flex-wrap gap-1">
                  <span
                    v-for="entry in path.acl"
                    :key="entry.principal"
                    class="rounded bg-slate-100 px-2 py-0.5 font-mono text-xs dark:bg-slate-800"
                  >
                    {{ entry.principal }}={{ entry.permission }}
                  </span>
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </StateBlock>

    <Pagination :page="page" :size="size" :total="total" @update:page="changePage" />

    <ModalDialog :open="createOpen" :title="t('accessPaths.create.title')" @close="closeCreate">
      <form id="create-access-path-form" class="space-y-4" @submit.prevent="submitCreate">
        <div>
          <label for="create-access-path" class="block text-sm font-medium">{{ t('accessPaths.create.path') }}</label>
          <input
            id="create-access-path"
            v-model="createPath"
            type="text"
            autocomplete="off"
            spellcheck="false"
            placeholder="project:/trunk"
            class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
          />
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('accessPaths.create.pathHint') }}</p>
        </div>

        <p
          v-if="createError"
          class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
          role="alert"
        >
          {{ createError }}
        </p>
      </form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <button
            type="button"
            class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
            :disabled="creating"
            @click="closeCreate"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="create-access-path-form"
            class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
            :disabled="creating"
          >
            {{ creating ? t('accessPaths.create.submitting') : t('accessPaths.create.submit') }}
          </button>
        </div>
      </template>
    </ModalDialog>
  </section>
</template>
