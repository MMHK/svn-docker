<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import { ApiError, createRepository, listRepositories } from '../api/client';
import { toError } from '../api/errors';
import type { Repository, RepositoryStructure } from '../api/types';
import DeleteRepositoryDialog from '../components/DeleteRepositoryDialog.vue';
import ModalDialog from '../components/ModalDialog.vue';
import Pagination from '../components/Pagination.vue';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';
import { useAuthStore } from '../stores/auth';
import { useToastStore } from '../stores/toast';

const auth = useAuthStore();
const toast = useToastStore();
const router = useRouter();

const items = ref<Repository[]>([]);
const page = ref(1);
const size = 20;
const total = ref(0);
const query = ref('');
const appliedQuery = ref('');
const loading = ref(false);
const error = ref<Error | null>(null);

const canCreate = computed(() => auth.can('repository:add'));
const canDelete = computed(() => auth.can('repository:delete'));

/**
 * Mirrors the API contract (plan section 5.4): 1-64 letters, digits, '_' or
 * '-' and no leading dot. The leading-dot rule is reported separately so the
 * operator sees the reason instead of a generic pattern failure.
 */
const REPOSITORY_NAME_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

// -- create ----------------------------------------------------------------

const createOpen = ref(false);
const createName = ref('');
const createStructure = ref<RepositoryStructure>('none');
const createProjectName = ref('');
const createAccessPath = ref(true);
const createError = ref('');
const creating = ref(false);

// -- delete ----------------------------------------------------------------

const deleteOpen = ref(false);
const deleteTarget = ref('');

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    const list = await listRepositories(page.value, size, appliedQuery.value);
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
  createName.value = '';
  createStructure.value = 'none';
  createProjectName.value = '';
  createAccessPath.value = true;
  createError.value = '';
  createOpen.value = true;
}

function closeCreate(): void {
  if (!creating.value) {
    createOpen.value = false;
  }
}

/** Client-side gate from the plan; the API repeats the checks and wins. */
function validateCreate(): string {
  const name = createName.value.trim();

  if (name === '') {
    return t('repos.create.nameRequired');
  }

  if (name.startsWith('.')) {
    return t('repos.create.nameLeadingDot');
  }

  if (!REPOSITORY_NAME_PATTERN.test(name)) {
    return t('repos.create.nameInvalid');
  }

  if (createStructure.value === 'multi') {
    const projectName = createProjectName.value.trim();

    if (projectName === '') {
      return t('repos.create.projectNameRequired');
    }

    if (projectName.startsWith('.') || !REPOSITORY_NAME_PATTERN.test(projectName)) {
      return t('repos.create.projectNameInvalid');
    }
  }

  return '';
}

function createErrorMessage(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('repos.create.failed');
  }

  switch (cause.status) {
    case 0:
      return t('common.networkErrorHint');
    case 403:
      return t('common.forbiddenHint');
    case 409:
      return t('repos.create.exists');
    case 422:
      return cause.message;
    default:
      return `${t('repos.create.failed')} (${cause.status})`;
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
    const created = await createRepository({
      name: createName.value.trim(),
      structure: createStructure.value,
      projectName: createStructure.value === 'multi' ? createProjectName.value.trim() : undefined,
      accessPath: createAccessPath.value,
    });

    createOpen.value = false;
    toast.push(t('repos.create.success'));
    await router.push({ name: 'repo-browse', params: { repo: created.name } });
  } catch (cause) {
    createError.value = createErrorMessage(cause);
  } finally {
    creating.value = false;
  }
}

function openDelete(name: string): void {
  deleteTarget.value = name;
  deleteOpen.value = true;
}

function closeDelete(): void {
  deleteOpen.value = false;
}

async function onDeleted(): Promise<void> {
  deleteOpen.value = false;
  await load();
}

onMounted(() => {
  void load();
});
</script>

<template>
  <section class="space-y-4">
    <div class="flex flex-wrap items-center gap-3">
      <h1 class="text-xl font-semibold">{{ t('repos.title') }}</h1>
      <button
        v-if="canCreate"
        type="button"
        class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
        @click="openCreate"
      >
        {{ t('repos.create.button') }}
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
        :placeholder="t('repos.searchPlaceholder')"
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
      :empty-text="t('repos.empty')"
    >
      <div class="overflow-x-auto rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <table class="w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('repos.name') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repos.parent') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repos.isRepository') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repos.headRevision') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repos.lastChangedAt') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('repos.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="repo in items" :key="repo.name" class="border-t border-slate-100 dark:border-slate-800">
              <td class="px-4 py-2">
                <RouterLink
                  :to="{ name: 'repo-browse', params: { repo: repo.name } }"
                  class="font-medium text-sky-700 hover:underline dark:text-sky-400"
                >
                  {{ repo.name }}
                </RouterLink>
              </td>
              <td class="px-4 py-2 font-mono text-xs text-slate-600 dark:text-slate-300">{{ repo.parent }}</td>
              <td class="px-4 py-2">{{ repo.isRepository ? t('common.yes') : t('common.no') }}</td>
              <td class="px-4 py-2">{{ repo.headRevision }}</td>
              <td class="px-4 py-2 text-slate-600 dark:text-slate-300">{{ repo.lastChangedAt ?? t('common.notAvailable') }}</td>
              <td class="px-4 py-2">
                <div class="flex flex-wrap gap-2">
                  <RouterLink
                    :to="{ name: 'repo-browse', params: { repo: repo.name } }"
                    class="text-xs text-sky-700 hover:underline dark:text-sky-400"
                  >
                    {{ t('repos.browse') }}
                  </RouterLink>
                  <button
                    v-if="canDelete"
                    type="button"
                    class="text-xs text-red-700 hover:underline dark:text-red-300"
                    @click="openDelete(repo.name)"
                  >
                    {{ t('repos.delete.button') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </StateBlock>

    <Pagination :page="page" :size="size" :total="total" @update:page="changePage" />

    <ModalDialog :open="createOpen" :title="t('repos.create.title')" @close="closeCreate">
      <form id="create-repository-form" class="space-y-4" @submit.prevent="submitCreate">
        <div>
          <label for="create-repository-name" class="block text-sm font-medium">{{ t('repos.create.name') }}</label>
          <input
            id="create-repository-name"
            v-model="createName"
            type="text"
            autocomplete="off"
            spellcheck="false"
            class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
          />
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('repos.create.nameHint') }}</p>
        </div>

        <div>
          <label for="create-repository-structure" class="block text-sm font-medium">
            {{ t('repos.create.structure') }}
          </label>
          <select
            id="create-repository-structure"
            v-model="createStructure"
            class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
          >
            <option value="none">{{ t('repos.create.structureNone') }}</option>
            <option value="single">{{ t('repos.create.structureSingle') }}</option>
            <option value="multi">{{ t('repos.create.structureMulti') }}</option>
          </select>
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('repos.create.structureHint') }}</p>
        </div>

        <div v-if="createStructure === 'multi'">
          <label for="create-repository-project" class="block text-sm font-medium">
            {{ t('repos.create.projectName') }}
          </label>
          <input
            id="create-repository-project"
            v-model="createProjectName"
            type="text"
            autocomplete="off"
            spellcheck="false"
            class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
          />
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('repos.create.projectNameHint') }}</p>
        </div>

        <label class="flex items-start gap-2 text-sm">
          <input v-model="createAccessPath" type="checkbox" class="mt-1" />
          <span>
            {{ t('repos.create.accessPath') }}
            <span class="mt-1 block text-xs text-slate-500 dark:text-slate-400">{{ t('repos.create.accessPathHint') }}</span>
          </span>
        </label>

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
            form="create-repository-form"
            class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
            :disabled="creating"
          >
            {{ creating ? t('repos.create.creating') : t('repos.create.submit') }}
          </button>
        </div>
      </template>
    </ModalDialog>

    <DeleteRepositoryDialog
      :open="deleteOpen"
      :name="deleteTarget"
      @close="closeDelete"
      @deleted="onDeleted"
    />
  </section>
</template>
