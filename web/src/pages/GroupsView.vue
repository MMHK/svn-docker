<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import { ApiError, createGroup, listGroups } from '../api/client';
import { toError } from '../api/errors';
import type { Group } from '../api/types';
import ModalDialog from '../components/ModalDialog.vue';
import Pagination from '../components/Pagination.vue';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';
import { useAuthStore } from '../stores/auth';
import { useToastStore } from '../stores/toast';

const auth = useAuthStore();
const toast = useToastStore();

const items = ref<Group[]>([]);
const page = ref(1);
const size = 20;
const total = ref(0);
const query = ref('');
const appliedQuery = ref('');
const loading = ref(false);
const error = ref<Error | null>(null);

/** Mirrors the authz group rule in api/pkg/model/group.go (plan section 5.4). */
const GROUP_NAME_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

const canCreate = computed(() => auth.can('group:add'));

const createOpen = ref(false);
const createName = ref('');
const createError = ref('');
const creating = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    const list = await listGroups(page.value, size, appliedQuery.value);
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
  createError.value = '';
  createOpen.value = true;
}

function closeCreate(): void {
  if (creating.value) {
    return;
  }

  createOpen.value = false;
}

/** Client-side checks from the plan; the API repeats them and stays authoritative. */
function validateCreate(): string {
  const name = createName.value.trim();

  if (name === '') {
    return t('groups.create.nameRequired');
  }

  if (!GROUP_NAME_PATTERN.test(name)) {
    return t('groups.create.nameInvalid');
  }

  return '';
}

function createErrorMessage(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('groups.create.failed');
  }

  switch (cause.status) {
    case 0:
      return t('common.networkErrorHint');
    case 403:
      return t('common.forbiddenHint');
    case 409:
      return t('groups.create.exists');
    case 422:
      return cause.message;
    default:
      return `${t('groups.create.failed')} (${cause.status})`;
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
    await createGroup(createName.value.trim());
    createOpen.value = false;
    toast.push(t('groups.create.success'));
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
      <h1 class="text-xl font-semibold">{{ t('groups.title') }}</h1>
      <button
        v-if="canCreate"
        type="button"
        class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
        @click="openCreate"
      >
        {{ t('groups.create.button') }}
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
        :placeholder="t('groups.searchPlaceholder')"
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
      :empty-text="t('groups.empty')"
    >
      <div class="overflow-x-auto rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <table class="w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('groups.name') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('groups.description') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('groups.memberCount') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="group in items" :key="group.name" class="border-t border-slate-100 dark:border-slate-800">
              <td class="px-4 py-2">
                <RouterLink
                  :to="{ name: 'group-detail', params: { name: group.name } }"
                  class="font-medium text-sky-700 hover:underline dark:text-sky-400"
                >
                  {{ group.name }}
                </RouterLink>
              </td>
              <td class="px-4 py-2 text-slate-600 dark:text-slate-300">
                {{ group.description === '' || group.description === undefined ? t('common.notAvailable') : group.description }}
              </td>
              <td class="px-4 py-2">{{ group.members.length }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </StateBlock>

    <Pagination :page="page" :size="size" :total="total" @update:page="changePage" />

    <ModalDialog :open="createOpen" :title="t('groups.create.title')" @close="closeCreate">
      <form id="create-group-form" class="space-y-4" @submit.prevent="submitCreate">
        <div>
          <label for="create-group-name" class="block text-sm font-medium">{{ t('groups.create.name') }}</label>
          <input
            id="create-group-name"
            v-model="createName"
            type="text"
            autocomplete="off"
            spellcheck="false"
            class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
          />
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('groups.create.nameHint') }}</p>
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
            form="create-group-form"
            class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
            :disabled="creating"
          >
            {{ creating ? t('groups.create.submitting') : t('groups.create.submit') }}
          </button>
        </div>
      </template>
    </ModalDialog>
  </section>
</template>
