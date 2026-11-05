<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { RouterLink, useRoute, useRouter } from 'vue-router';
import { getRepositoryInfo, getRepositoryTree } from '../api/client';
import { describeError, toError } from '../api/errors';
import type { Repository, RepositoryEntry, RepositoryTree } from '../api/types';
import DeleteRepositoryDialog from '../components/DeleteRepositoryDialog.vue';
import Pagination from '../components/Pagination.vue';
import StateBlock from '../components/StateBlock.vue';
import { locale, t } from '../i18n';

/**
 * Repository browser (plan T4.2/T4.6).
 *
 * Route: /repositories/:repo/browse?path=&rev=&page= — `path` is relative to
 * the repository root, an empty `rev` means HEAD. The API validates both, so an
 * illegal value renders its 422 message instead of a blank page.
 */
const props = defineProps<{ repo: string }>();

const route = useRoute();
const router = useRouter();

const PAGE_SIZE = 100;

function queryString(key: string): string {
  const value = route.query[key];

  return typeof value === 'string' ? value : '';
}

const path = computed(() => queryString('path'));
const revParam = computed(() => queryString('rev'));
const pageParam = computed(() => {
  const raw = queryString('page');

  return /^[1-9][0-9]*$/.test(raw) ? Number(raw) : 1;
});

// -- repository info -------------------------------------------------------

const info = ref<Repository | null>(null);
const infoLoading = ref(false);
const infoError = ref<Error | null>(null);
let infoRequestId = 0;

const infoDescribed = computed(() => (infoError.value === null ? null : describeError(infoError.value)));

async function loadInfo(): Promise<void> {
  const current = (infoRequestId += 1);

  infoLoading.value = true;
  infoError.value = null;

  try {
    const result = await getRepositoryInfo(props.repo);

    if (current !== infoRequestId) {
      return;
    }

    info.value = result;
  } catch (cause) {
    if (current !== infoRequestId) {
      return;
    }

    info.value = null;
    infoError.value = toError(cause);
  } finally {
    if (current === infoRequestId) {
      infoLoading.value = false;
    }
  }
}

// -- tree ------------------------------------------------------------------

const tree = ref<RepositoryTree | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);
let requestId = 0;

async function loadTree(): Promise<void> {
  const current = (requestId += 1);

  loading.value = true;
  error.value = null;

  try {
    const result = await getRepositoryTree(
      props.repo,
      path.value,
      revParam.value === '' ? undefined : revParam.value,
      pageParam.value,
      PAGE_SIZE,
    );

    if (current !== requestId) {
      return;
    }

    tree.value = result;

    // A hand-edited page beyond the last page would otherwise look like an
    // empty directory; move to the last real page instead.
    const lastPage = Math.max(1, Math.ceil(result.total / Math.max(1, result.size)));
    if (result.total > 0 && result.entries.length === 0 && pageParam.value > lastPage) {
      void router.replace({
        name: 'repo-browse',
        params: { repo: props.repo },
        query: browseQuery(path.value, revParam.value, lastPage),
      });
    }
  } catch (cause) {
    if (current !== requestId) {
      return;
    }

    tree.value = null;
    error.value = toError(cause);
  } finally {
    if (current === requestId) {
      loading.value = false;
    }
  }
}

// -- navigation ------------------------------------------------------------

interface Crumb {
  label: string;
  path: string;
}

const crumbs = computed<Crumb[]>(() => {
  if (path.value === '') {
    return [];
  }

  const segments = path.value.split('/');

  return segments.map((segment, index) => ({
    label: segment,
    path: segments.slice(0, index + 1).join('/'),
  }));
});

/** null at the repository root; the empty string is the root path itself. */
const parentPath = computed<string | null>(() => {
  if (path.value === '') {
    return null;
  }

  return path.value.split('/').slice(0, -1).join('/');
});

function browseQuery(targetPath: string, rev: string, page: number): Record<string, string> {
  const query: Record<string, string> = {};

  if (targetPath !== '') {
    query.path = targetPath;
  }

  if (rev !== '') {
    query.rev = rev;
  }

  if (page > 1) {
    query.page = String(page);
  }

  return query;
}

function navigateTo(targetPath: string, targetPage = 1): void {
  void router.push({
    name: 'repo-browse',
    params: { repo: props.repo },
    query: browseQuery(targetPath, revParam.value, targetPage),
  });
}

function openEntry(entry: RepositoryEntry): void {
  if (entry.kind !== 'dir') {
    return;
  }

  const next = path.value === '' ? entry.name : `${path.value}/${entry.name}`;
  navigateTo(next);
}

/** The row is clickable for directories; links keep their own navigation. */
function onRowClick(event: MouseEvent, entry: RepositoryEntry): void {
  if ((event.target as HTMLElement | null)?.closest('a') !== null) {
    return;
  }

  openEntry(entry);
}

function changePage(next: number): void {
  navigateTo(path.value, next);
}

function goUp(): void {
  if (parentPath.value !== null) {
    navigateTo(parentPath.value);
  }
}

// -- revision input --------------------------------------------------------

const revInput = ref(revParam.value);
const revError = ref('');
const revValid = /^[1-9][0-9]*$/;

watch(revParam, (value) => {
  revInput.value = value;
  revError.value = '';
});

function applyRevision(): void {
  const raw = revInput.value.trim();

  if (raw !== '' && !revValid.test(raw)) {
    revError.value = t('repoBrowse.revInvalid');

    return;
  }

  revError.value = '';

  if (raw === revParam.value) {
    void loadTree();

    return;
  }

  void router.push({
    name: 'repo-browse',
    params: { repo: props.repo },
    query: browseQuery(path.value, raw, 1),
  });
}

// -- presentation ----------------------------------------------------------

function formatDate(value?: string): string {
  if (value === undefined || value === '') {
    return t('common.notAvailable');
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString(locale.value === 'zh_CN' ? 'zh-TW' : 'en-US');
}

function formatSize(entry: RepositoryEntry): string {
  if (entry.kind !== 'file') {
    return t('common.notAvailable');
  }

  return `${entry.size ?? 0} B`;
}

// -- delete ----------------------------------------------------------------

const deleteOpen = ref(false);

function closeDelete(): void {
  deleteOpen.value = false;
}

async function onDeleted(): Promise<void> {
  deleteOpen.value = false;
  await router.push({ name: 'repos' });
}

watch(
  () => props.repo,
  () => {
    void loadInfo();
  },
  { immediate: true },
);

watch(
  () => [props.repo, path.value, revParam.value, pageParam.value],
  () => {
    void loadTree();
  },
  { immediate: true },
);
</script>

<template>
  <section class="space-y-4">
    <RouterLink :to="{ name: 'repos' }" class="text-sm text-sky-700 hover:underline dark:text-sky-400">
      ← {{ t('repos.title') }}
    </RouterLink>

    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <h1 class="text-xl font-semibold">{{ repo }}</h1>

        <div v-if="infoLoading" class="mt-1 text-xs text-slate-500 dark:text-slate-400" role="status">
          {{ t('repoBrowse.loadingInfo') }}
        </div>
        <div v-else-if="info" class="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-600 dark:text-slate-300">
          <span>
            {{ t('repoBrowse.head') }}:
            <span class="font-mono font-medium">{{ info.headRevision }}</span>
          </span>
          <span>
            {{ t('repoBrowse.lastChanged') }}:
            <span class="font-mono">r{{ info.lastChangedRev }}</span>
            <template v-if="info.lastChangedAuthor">
              · {{ info.lastChangedAuthor }}
            </template>
            <template v-if="info.lastChangedAt">
              · {{ formatDate(info.lastChangedAt) }}
            </template>
          </span>
        </div>
        <p v-else-if="infoDescribed" class="mt-1 text-xs text-amber-700 dark:text-amber-300">
          {{ t('repoBrowse.infoUnavailable') }} — {{ infoDescribed.title }}: {{ infoDescribed.detail }}
        </p>
      </div>

      <div class="flex flex-wrap gap-2">
        <button
          type="button"
          class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
          :disabled="loading"
          @click="loadTree"
        >
          {{ t('common.refresh') }}
        </button>
        <button
          type="button"
          class="rounded border border-red-300 px-3 py-1 text-sm text-red-700 hover:bg-red-50 dark:border-red-900 dark:text-red-300 dark:hover:bg-red-950"
          @click="deleteOpen = true"
        >
          {{ t('repos.delete.button') }}
        </button>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2 text-sm">
      <button
        v-if="parentPath !== null"
        type="button"
        class="rounded border border-slate-300 px-2 py-1 hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
        @click="goUp"
      >
        ↑ {{ t('repoBrowse.up') }}
      </button>

      <nav class="flex flex-wrap items-center gap-1" :aria-label="t('repoBrowse.breadcrumb')">
        <RouterLink
          :to="{ name: 'repo-browse', params: { repo }, query: browseQuery('', revParam, 1) }"
          class="rounded px-1 text-sky-700 hover:underline dark:text-sky-400"
        >
          {{ repo }}
        </RouterLink>
        <template v-for="crumb in crumbs" :key="crumb.path">
          <span class="text-slate-400 dark:text-slate-500">/</span>
          <RouterLink
            :to="{ name: 'repo-browse', params: { repo }, query: browseQuery(crumb.path, revParam, 1) }"
            class="rounded px-1 text-sky-700 hover:underline dark:text-sky-400"
          >
            {{ crumb.label }}
          </RouterLink>
        </template>
      </nav>
    </div>

    <form class="flex flex-wrap items-end gap-2" @submit.prevent="applyRevision">
      <div>
        <label for="repository-revision" class="block text-xs font-medium text-slate-600 dark:text-slate-300">
          {{ t('repoBrowse.revision') }}
        </label>
        <input
          id="repository-revision"
          v-model="revInput"
          type="text"
          inputmode="numeric"
          autocomplete="off"
          spellcheck="false"
          :placeholder="t('repoBrowse.revisionPlaceholder')"
          class="mt-1 w-40 rounded border border-slate-300 bg-white px-3 py-1 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
        />
      </div>
      <button
        type="submit"
        class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
      >
        {{ t('repoBrowse.go') }}
      </button>
      <span v-if="tree" class="pb-1 text-xs text-slate-600 dark:text-slate-300">
        {{ t('repoBrowse.currentRevision') }}:
        <span class="font-mono font-medium">{{ tree.revision }}</span>
      </span>
    </form>

    <p
      v-if="revError"
      class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
      role="alert"
    >
      {{ revError }}
    </p>

    <StateBlock
      :loading="loading"
      :error="error"
      :empty="tree !== null && tree.entries.length === 0"
      :empty-text="t('repoBrowse.emptyDirectory')"
    >
      <div class="overflow-x-auto rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <table class="w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('repoBrowse.name') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repoBrowse.kind') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repoBrowse.size') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repoBrowse.revision') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repoBrowse.author') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repoBrowse.date') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="entry in tree?.entries ?? []"
              :key="entry.name"
              class="border-t border-slate-100 dark:border-slate-800"
              :class="{ 'cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50': entry.kind === 'dir' }"
              @click="onRowClick($event, entry)"
            >
              <td class="px-4 py-2">
                <RouterLink
                  v-if="entry.kind === 'dir'"
                  :to="{
                    name: 'repo-browse',
                    params: { repo },
                    query: browseQuery(path === '' ? entry.name : `${path}/${entry.name}`, revParam, 1),
                  }"
                  class="font-medium text-sky-700 hover:underline dark:text-sky-400"
                >
                  {{ entry.name }}
                </RouterLink>
                <span v-else class="font-medium">{{ entry.name }}</span>
              </td>
              <td class="px-4 py-2">{{ entry.kind === 'dir' ? t('repoBrowse.dir') : t('repoBrowse.file') }}</td>
              <td class="px-4 py-2 text-slate-600 dark:text-slate-300">{{ formatSize(entry) }}</td>
              <td class="px-4 py-2 font-mono text-xs">r{{ entry.revision }}</td>
              <td class="px-4 py-2 text-slate-600 dark:text-slate-300">{{ entry.author ?? t('common.notAvailable') }}</td>
              <td class="px-4 py-2 text-slate-600 dark:text-slate-300">{{ formatDate(entry.date) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </StateBlock>

    <Pagination
      v-if="tree !== null"
      :page="pageParam"
      :size="tree.size"
      :total="tree.total"
      @update:page="changePage"
    />

    <DeleteRepositoryDialog :open="deleteOpen" :name="repo" @close="closeDelete" @deleted="onDeleted" />
  </section>
</template>
