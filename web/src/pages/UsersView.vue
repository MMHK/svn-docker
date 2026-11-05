<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import { ApiError, createUser, listUsers } from '../api/client';
import { toError } from '../api/errors';
import type { User } from '../api/types';
import ModalDialog from '../components/ModalDialog.vue';
import Pagination from '../components/Pagination.vue';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';
import { useToastStore } from '../stores/toast';

const toast = useToastStore();
const router = useRouter();

const items = ref<User[]>([]);
const page = ref(1);
const size = 20;
const total = ref(0);
const query = ref('');
const appliedQuery = ref('');
const loading = ref(false);
const error = ref<Error | null>(null);

/** Mirrors the API's account-name rule so obvious mistakes never hit the wire. */
const USERNAME_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/;
const MAX_PASSWORD_LENGTH = 256;

const createOpen = ref(false);
const createUsername = ref('');
const createPassword = ref('');
const createConfirm = ref('');
const createError = ref('');
const creating = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    const list = await listUsers(page.value, size, appliedQuery.value);
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

/** The whole row opens the detail page; the name link keeps it keyboard usable. */
function openUser(event: MouseEvent, name: string): void {
  if ((event.target as HTMLElement | null)?.closest('a') !== null) {
    return;
  }

  void router.push({ name: 'user-detail', params: { name } });
}

function openCreate(): void {
  createUsername.value = '';
  createPassword.value = '';
  createConfirm.value = '';
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
  const username = createUsername.value.trim();

  if (username === '') {
    return t('users.create.usernameRequired');
  }

  if (!USERNAME_PATTERN.test(username)) {
    return t('users.create.usernameInvalid');
  }

  if (createPassword.value === '') {
    return t('users.create.passwordRequired');
  }

  if (createPassword.value.length > MAX_PASSWORD_LENGTH) {
    return t('users.create.passwordTooLong');
  }

  if (/[\r\n]/.test(createPassword.value)) {
    return t('users.create.passwordNewline');
  }

  if (createPassword.value !== createConfirm.value) {
    return t('users.create.passwordMismatch');
  }

  return '';
}

function createErrorMessage(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('users.create.failed');
  }

  switch (cause.status) {
    case 0:
      return t('common.networkErrorHint');
    case 403:
      return t('common.forbiddenHint');
    case 409:
      return t('users.create.exists');
    case 422:
      return cause.message;
    default:
      return `${t('users.create.failed')} (${cause.status})`;
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
    await createUser(createUsername.value.trim(), createPassword.value);
    createOpen.value = false;
    toast.push(t('users.create.success'));
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
      <h1 class="text-xl font-semibold">{{ t('users.title') }}</h1>
      <button
        type="button"
        class="rounded-md bg-[#1f883d] px-3 py-1 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:opacity-50"
        @click="openCreate"
      >
        {{ t('users.create.button') }}
      </button>
      <button
        type="button"
        class="rounded-md border border-[#d0d7de] px-3 py-1 text-sm hover:bg-[#f6f8fa] disabled:opacity-50 dark:border-[#30363d] dark:hover:bg-[#21262d]"
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
        :placeholder="t('users.searchPlaceholder')"
        :aria-label="t('common.search')"
        class="w-64 rounded-md border border-[#d0d7de] bg-[#ffffff] px-3 py-1 text-sm dark:border-[#30363d] dark:bg-[#21262d]"
      />
      <button
        type="submit"
        class="rounded-md border border-[#d0d7de] px-3 py-1 text-sm hover:bg-[#f6f8fa] dark:border-[#30363d] dark:hover:bg-[#21262d]"
      >
        {{ t('common.search') }}
      </button>
    </form>

    <StateBlock
      :loading="loading"
      :error="error"
      :empty="items.length === 0"
      :empty-text="t('users.empty')"
    >
      <div class="overflow-x-auto rounded-md border border-[#d0d7de] bg-[#ffffff] dark:border-[#30363d] dark:bg-[#161b22]">
        <table class="w-full text-left text-sm">
          <thead class="bg-[#f6f8fa] text-xs uppercase tracking-wide text-[#656d76] dark:bg-[#161b22] dark:text-[#8b949e]">
            <tr>
              <th class="px-4 py-2 font-medium">{{ t('users.name') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('users.comment') }}</th>
              <th class="px-4 py-2 font-medium">{{ t('users.groups') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="user in items"
              :key="user.name"
              class="cursor-pointer border-t border-[#eaeef2] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:hover:bg-[#21262d]"
              @click="openUser($event, user.name)"
            >
              <td class="px-4 py-2">
                <RouterLink
                  :to="{ name: 'user-detail', params: { name: user.name } }"
                  class="font-medium text-[#0969da] hover:underline dark:text-[#2f81f7]"
                >
                  {{ user.name }}
                </RouterLink>
              </td>
              <td class="px-4 py-2 text-[#656d76] dark:text-[#c9d1d9]">{{ user.comment ?? t('common.notAvailable') }}</td>
              <td class="px-4 py-2">
                <span v-if="user.groups.length === 0" class="text-[#656d76] dark:text-[#6e7681]">{{ t('common.none') }}</span>
                <span v-else class="flex flex-wrap gap-1">
                  <span
                    v-for="group in user.groups"
                    :key="group"
                    class="rounded-md bg-[#f6f8fa] px-2 py-0.5 text-xs dark:bg-[#21262d]"
                  >
                    {{ group }}
                  </span>
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </StateBlock>

    <Pagination :page="page" :size="size" :total="total" @update:page="changePage" />

    <ModalDialog :open="createOpen" :title="t('users.create.title')" @close="closeCreate">
      <form id="create-user-form" class="space-y-4" @submit.prevent="submitCreate">
        <div>
          <label for="create-username" class="block text-sm font-medium">{{ t('users.create.username') }}</label>
          <input
            id="create-username"
            v-model="createUsername"
            type="text"
            autocomplete="off"
            spellcheck="false"
            class="mt-1 w-full rounded-md border border-[#d0d7de] bg-[#ffffff] px-3 py-2 text-sm dark:border-[#30363d] dark:bg-[#21262d]"
          />
          <p class="mt-1 text-xs text-[#656d76] dark:text-[#8b949e]">{{ t('users.create.usernameHint') }}</p>
        </div>

        <div>
          <label for="create-password" class="block text-sm font-medium">{{ t('users.create.password') }}</label>
          <input
            id="create-password"
            v-model="createPassword"
            type="password"
            autocomplete="new-password"
            class="mt-1 w-full rounded-md border border-[#d0d7de] bg-[#ffffff] px-3 py-2 text-sm dark:border-[#30363d] dark:bg-[#21262d]"
          />
        </div>

        <div>
          <label for="create-confirm" class="block text-sm font-medium">{{ t('users.create.confirmPassword') }}</label>
          <input
            id="create-confirm"
            v-model="createConfirm"
            type="password"
            autocomplete="new-password"
            class="mt-1 w-full rounded-md border border-[#d0d7de] bg-[#ffffff] px-3 py-2 text-sm dark:border-[#30363d] dark:bg-[#21262d]"
          />
        </div>

        <p
          v-if="createError"
          class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
          role="alert"
        >
          {{ createError }}
        </p>
      </form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <button
            type="button"
            class="rounded-md border border-[#d0d7de] px-3 py-1 text-sm hover:bg-[#f6f8fa] disabled:opacity-50 dark:border-[#30363d] dark:hover:bg-[#21262d]"
            :disabled="creating"
            @click="closeCreate"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="create-user-form"
            class="rounded-md bg-[#1f883d] px-3 py-1 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:opacity-50"
            :disabled="creating"
          >
            {{ creating ? t('users.create.creating') : t('users.create.submit') }}
          </button>
        </div>
      </template>
    </ModalDialog>
  </section>
</template>
