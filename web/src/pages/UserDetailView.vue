<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import {
  changeUserPassword,
  deleteUser,
  destroySession,
  getUser,
  getUserAccessPaths,
  listGroups,
  setUserGroups,
} from '../api/client';
import { encodeAccessPath } from '../api/encoding';
import { describeError, toError } from '../api/errors';
import type { Group, User, UserAccessPath } from '../api/types';
import ModalDialog from '../components/ModalDialog.vue';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';
import { useAuthStore } from '../stores/auth';
import { useToastStore } from '../stores/toast';

const props = defineProps<{ name: string }>();

const auth = useAuthStore();
const toast = useToastStore();
const router = useRouter();

const MAX_PASSWORD_LENGTH = 256;

const user = ref<User | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);

const accessPaths = ref<UserAccessPath[]>([]);
const accessLoading = ref(false);
const accessError = ref<Error | null>(null);

const canViewAccessPaths = computed(() => true);
const isSelf = computed(() => user.value !== null && user.value.name.toLowerCase() === auth.userName.toLowerCase());
const canChangePassword = computed(() => true);

// -- change password -------------------------------------------------------

const passwordOpen = ref(false);
const newPassword = ref('');
const confirmPassword = ref('');
const passwordError = ref('');
const passwordSaving = ref(false);

// -- delete ----------------------------------------------------------------

const deleteOpen = ref(false);
const deleteConfirm = ref('');
const deleteError = ref('');
const deleteSaving = ref(false);

// -- groups ----------------------------------------------------------------

const groupsOpen = ref(false);
const groupOptions = ref<Group[]>([]);
const selectedGroups = ref<string[]>([]);
const groupsLoading = ref(false);
const groupsSaving = ref(false);
const groupsError = ref('');

interface AccessRow {
  path: string;
  permission: string;
  source: 'user' | 'group' | 'global';
  group: string;
}

/** The API already merges direct, `@group` and `*` sources. */
const accessRows = computed<AccessRow[]>(() =>
  accessPaths.value.map((item) => ({
    path: item.path,
    permission: item.permission,
    source: item.source,
    group: item.group ?? '',
  })),
);

/** One-line rendering of any API failure, including `operation_failed` steps. */
function actionErrorText(cause: unknown): string {
  const described = describeError(toError(cause));

  return `${described.title}: ${described.detail}`;
}

async function loadUser(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    user.value = await getUser(props.name);
  } catch (cause) {
    user.value = null;
    error.value = toError(cause);
  } finally {
    loading.value = false;
  }
}

async function loadAccessPaths(): Promise<void> {
  if (!canViewAccessPaths.value) {
    return;
  }

  accessLoading.value = true;
  accessError.value = null;

  try {
    const result = await getUserAccessPaths(props.name);
    accessPaths.value = result.items ?? [];
  } catch (cause) {
    accessPaths.value = [];
    accessError.value = toError(cause);
  } finally {
    accessLoading.value = false;
  }
}

// -- change password -------------------------------------------------------

function openPassword(): void {
  newPassword.value = '';
  confirmPassword.value = '';
  passwordError.value = '';
  passwordOpen.value = true;
}

function closePassword(): void {
  if (!passwordSaving.value) {
    passwordOpen.value = false;
  }
}

/** Shared client-side checks (required, max 256, no newline, confirmation). */
function validateNewPassword(): string {
  if (newPassword.value === '') {
    return t('userDetail.passwordRequired');
  }

  if (newPassword.value.length > MAX_PASSWORD_LENGTH) {
    return t('userDetail.passwordTooLong');
  }

  if (/[\r\n]/.test(newPassword.value)) {
    return t('userDetail.passwordNewline');
  }

  if (newPassword.value !== confirmPassword.value) {
    return t('userDetail.passwordMismatch');
  }

  return '';
}

async function submitPassword(): Promise<void> {
  const current = user.value;

  if (current === null || passwordSaving.value) {
    return;
  }

  passwordError.value = validateNewPassword();
  if (passwordError.value !== '') {
    return;
  }

  passwordSaving.value = true;

  try {
    await changeUserPassword(current.name, newPassword.value);
    newPassword.value = '';
    confirmPassword.value = '';
    passwordOpen.value = false;

    if (isSelf.value) {
      // The API revoked every token of this account, including ours (plan
      // section 5.7), so finish the sign-out flow locally as well.
      toast.push(t('userDetail.passwordSelfSuccess'));

      try {
        await destroySession();
      } catch {
        // Already revoked; clearing the client state is enough.
      }

      auth.clear();
      await router.push({ name: 'login' });
    } else {
      toast.push(t('userDetail.passwordSuccess'));
    }
  } catch (cause) {
    passwordError.value = actionErrorText(cause);
  } finally {
    passwordSaving.value = false;
  }
}

// -- delete ----------------------------------------------------------------

function openDelete(): void {
  deleteConfirm.value = '';
  deleteError.value = '';
  deleteOpen.value = true;
}

function closeDelete(): void {
  if (!deleteSaving.value) {
    deleteOpen.value = false;
  }
}

async function submitDelete(): Promise<void> {
  const current = user.value;

  if (current === null || deleteSaving.value) {
    return;
  }

  // Typing the exact account name is the confirmation gate (plan D7 pattern).
  if (deleteConfirm.value.trim() !== current.name) {
    return;
  }

  deleteSaving.value = true;
  deleteError.value = '';

  try {
    await deleteUser(current.name);
    deleteOpen.value = false;
    toast.push(t('userDetail.deleteSuccess'));
    await router.push({ name: 'users' });
  } catch (cause) {
    deleteError.value = actionErrorText(cause);
  } finally {
    deleteSaving.value = false;
  }
}

// -- groups ----------------------------------------------------------------

async function openGroups(): Promise<void> {
  const current = user.value;

  if (current === null) {
    return;
  }

  groupsError.value = '';
  selectedGroups.value = [...current.groups];
  groupsOpen.value = true;
  groupsLoading.value = true;

  try {
    const list = await listGroups(1, 500);
    groupOptions.value = list.items ?? [];
  } catch (cause) {
    groupOptions.value = [];
    groupsError.value = actionErrorText(cause);
  } finally {
    groupsLoading.value = false;
  }
}

function closeGroups(): void {
  if (!groupsSaving.value) {
    groupsOpen.value = false;
  }
}

async function submitGroups(): Promise<void> {
  const current = user.value;

  if (current === null || groupsSaving.value) {
    return;
  }

  groupsSaving.value = true;
  groupsError.value = '';

  try {
    const updated = await setUserGroups(current.name, selectedGroups.value);
    user.value = { ...current, groups: updated.groups ?? [] };
    groupsOpen.value = false;
    toast.push(t('userDetail.groupsSaved'));
  } catch (cause) {
    groupsError.value = actionErrorText(cause);
  } finally {
    groupsSaving.value = false;
  }
}

watch(
  () => props.name,
  () => {
    accessPaths.value = [];
    passwordOpen.value = false;
    deleteOpen.value = false;
    groupsOpen.value = false;
    void loadUser();
    void loadAccessPaths();
  },
  { immediate: true },
);
</script>

<template>
  <section class="space-y-6">
    <RouterLink :to="{ name: 'users' }" class="text-sm text-sky-700 hover:underline dark:text-sky-400">
      ← {{ t('users.title') }}
    </RouterLink>

    <StateBlock :loading="loading" :error="error" :empty="user === null" :empty-text="t('userDetail.empty')">
      <template v-if="user">
        <div>
          <h1 class="text-xl font-semibold">{{ user.name }}</h1>
          <p v-if="user.comment" class="mt-1 text-sm text-slate-600 dark:text-slate-300">{{ user.comment }}</p>

          <div class="mt-3 flex flex-wrap gap-2">
            <button
              v-if="canChangePassword"
              type="button"
              class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
              @click="openPassword"
            >
              {{ t('userDetail.changePassword') }}
            </button>
            <button
              type="button"
              class="rounded border border-red-300 px-3 py-1 text-sm text-red-700 hover:bg-red-50 dark:border-red-900 dark:text-red-300 dark:hover:bg-red-950"
              @click="openDelete"
            >
              {{ t('userDetail.delete') }}
            </button>
          </div>
        </div>

        <div class="mt-4 grid gap-4 sm:grid-cols-2">
          <div class="rounded border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
            <div class="flex items-center justify-between gap-2">
              <h2 class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">{{ t('users.groups') }}</h2>
              <button
                type="button"
                class="text-xs text-sky-700 hover:underline dark:text-sky-400"
                @click="openGroups"
              >
                {{ t('userDetail.editGroups') }}
              </button>
            </div>
            <p v-if="user.groups.length === 0" class="mt-2 text-sm text-slate-500 dark:text-slate-400">
              {{ t('common.none') }}
            </p>
            <ul v-else class="mt-2 flex flex-wrap gap-2">
              <li v-for="group in user.groups" :key="group" class="rounded bg-slate-100 px-2 py-1 text-xs dark:bg-slate-800">
                <RouterLink :to="{ name: 'group-detail', params: { name: group } }" class="hover:underline">
                  {{ group }}
                </RouterLink>
              </li>
            </ul>
          </div>
        </div>

        <div class="mt-6">
          <h2 class="text-sm font-semibold">{{ t('userDetail.access') }}</h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('userDetail.accessNote') }}</p>

          <p v-if="!canViewAccessPaths" class="mt-3 rounded border border-slate-200 bg-white px-4 py-3 text-sm text-slate-500 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-400">
            {{ t('userDetail.accessForbidden') }}
          </p>
          <p v-else-if="accessLoading" class="mt-3 text-sm text-slate-500 dark:text-slate-400">
            {{ t('common.loading') }}
          </p>
          <div
            v-else-if="accessError"
            class="mt-3 rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
            role="alert"
          >
            <p class="font-medium">{{ describeError(accessError).title }}</p>
            <p class="mt-1">{{ describeError(accessError).detail }}</p>
          </div>
          <p
            v-else-if="accessRows.length === 0"
            class="mt-3 rounded border border-dashed border-slate-300 bg-white px-4 py-6 text-center text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-400"
          >
            {{ t('userDetail.emptyAccess') }}
          </p>
          <div v-else class="mt-3 overflow-x-auto rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
            <table class="w-full text-left text-sm">
              <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
                <tr>
                  <th class="px-4 py-2 font-medium">{{ t('accessPaths.path') }}</th>
                  <th class="px-4 py-2 font-medium">{{ t('accessPaths.permission') }}</th>
                  <th class="px-4 py-2 font-medium">{{ t('userDetail.source') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in accessRows" :key="`${row.path}-${row.source}-${row.group}`" class="border-t border-slate-100 dark:border-slate-800">
                  <td class="px-4 py-2 font-mono text-xs">
                    <RouterLink
                      :to="{ name: 'access-path-detail', params: { path: encodeAccessPath(row.path) } }"
                      class="text-sky-700 hover:underline dark:text-sky-400"
                    >
                      {{ row.path }}
                    </RouterLink>
                  </td>
                  <td class="px-4 py-2 font-mono text-xs">{{ row.permission }}</td>
                  <td class="px-4 py-2 text-slate-600 dark:text-slate-300">
                    <span v-if="row.source === 'user'">{{ t('userDetail.direct') }}</span>
                    <span v-else-if="row.source === 'group'">{{ t('userDetail.viaGroup') }} @{{ row.group }}</span>
                    <span v-else>{{ t('userDetail.global') }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <ModalDialog :open="passwordOpen" :title="t('userDetail.passwordTitle')" @close="closePassword">
          <form id="change-password-form" class="space-y-4" @submit.prevent="submitPassword">
            <p class="text-xs text-slate-500 dark:text-slate-400">
              {{ isSelf ? t('userDetail.passwordSelfHint') : t('userDetail.passwordOtherHint') }}
            </p>

            <div>
              <label for="new-password" class="block text-sm font-medium">{{ t('userDetail.newPassword') }}</label>
              <input
                id="new-password"
                v-model="newPassword"
                type="password"
                autocomplete="new-password"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              />
            </div>

            <div>
              <label for="confirm-password" class="block text-sm font-medium">{{ t('userDetail.confirmPassword') }}</label>
              <input
                id="confirm-password"
                v-model="confirmPassword"
                type="password"
                autocomplete="new-password"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              />
            </div>

            <p
              v-if="passwordError"
              class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
              role="alert"
            >
              {{ passwordError }}
            </p>
          </form>

          <template #footer>
            <div class="flex justify-end gap-2">
              <button
                type="button"
                class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
                :disabled="passwordSaving"
                @click="closePassword"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="submit"
                form="change-password-form"
                class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
                :disabled="passwordSaving"
              >
                {{ passwordSaving ? t('userDetail.passwordSubmitting') : t('userDetail.passwordSubmit') }}
              </button>
            </div>
          </template>
        </ModalDialog>

        <ModalDialog :open="deleteOpen" :title="t('userDetail.deleteTitle')" @close="closeDelete">
          <form id="delete-user-form" class="space-y-4" @submit.prevent="submitDelete">
            <p class="text-sm text-red-700 dark:text-red-300">{{ t('userDetail.deleteHint') }}</p>

            <p class="text-sm">
              {{ t('userDetail.deleteConfirmLabel') }}
              <span class="font-mono font-semibold">{{ user.name }}</span>
            </p>

            <div>
              <label for="delete-confirm" class="block text-sm font-medium">{{ t('userDetail.deleteConfirmField') }}</label>
              <input
                id="delete-confirm"
                v-model="deleteConfirm"
                type="text"
                autocomplete="off"
                spellcheck="false"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
              />
            </div>

            <p
              v-if="deleteError"
              class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
              role="alert"
            >
              {{ deleteError }}
            </p>
          </form>

          <template #footer>
            <div class="flex justify-end gap-2">
              <button
                type="button"
                class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
                :disabled="deleteSaving"
                @click="closeDelete"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="submit"
                form="delete-user-form"
                class="rounded bg-red-600 px-3 py-1 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
                :disabled="deleteSaving || deleteConfirm.trim() !== user.name"
              >
                {{ deleteSaving ? t('userDetail.deleting') : t('userDetail.deleteSubmit') }}
              </button>
            </div>
          </template>
        </ModalDialog>

        <ModalDialog :open="groupsOpen" :title="t('userDetail.groupsTitle')" @close="closeGroups">
          <form id="edit-groups-form" class="space-y-3" @submit.prevent="submitGroups">
            <p v-if="groupsLoading" class="text-sm text-slate-500 dark:text-slate-400">{{ t('common.loading') }}</p>
            <p v-else-if="groupOptions.length === 0" class="text-sm text-slate-500 dark:text-slate-400">
              {{ t('userDetail.noGroups') }}
            </p>
            <div v-else class="space-y-2">
              <label
                v-for="group in groupOptions"
                :key="group.name"
                class="flex items-start gap-2 rounded border border-slate-200 px-3 py-2 text-sm dark:border-slate-800"
              >
                <input v-model="selectedGroups" type="checkbox" :value="group.name" class="mt-0.5" />
                <span>
                  <span class="font-medium">{{ group.name }}</span>
                  <span class="block text-xs text-slate-500 dark:text-slate-400">
                    {{ group.members.length }} {{ t('userDetail.memberCount') }}
                  </span>
                </span>
              </label>
            </div>

            <p
              v-if="groupsError"
              class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
              role="alert"
            >
              {{ groupsError }}
            </p>
          </form>

          <template #footer>
            <div class="flex justify-end gap-2">
              <button
                type="button"
                class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
                :disabled="groupsSaving"
                @click="closeGroups"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="submit"
                form="edit-groups-form"
                class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
                :disabled="groupsSaving || groupsLoading"
              >
                {{ groupsSaving ? t('userDetail.saving') : t('common.save') }}
              </button>
            </div>
          </template>
        </ModalDialog>
      </template>
    </StateBlock>
  </section>
</template>
