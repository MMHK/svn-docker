<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import {
  ApiError,
  deleteAccessPath,
  deleteAccessPathPermission,
  getAccessPath,
  listGroups,
  listUsers,
  setAccessPathManagers,
  setAccessPathPermission,
} from '../api/client';
import { decodeAccessPath } from '../api/encoding';
import { describeError, toError } from '../api/errors';
import type { ACLEntry, AccessPath, Permission } from '../api/types';
import ModalDialog from '../components/ModalDialog.vue';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';
import { useAuthStore } from '../stores/auth';
import { useToastStore } from '../stores/toast';

const props = defineProps<{ path: string }>();

const auth = useAuthStore();
const toast = useToastStore();
const router = useRouter();

const decoded = computed(() => decodeAccessPath(props.path));

const accessPath = ref<AccessPath | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);
const forbidden = ref(false);

const canAssign = computed(() => auth.can('access-path:assign'));
const canUnassign = computed(() => auth.can('access-path:unassign'));
const canDelete = computed(() => auth.can('access-path:delete'));
const canEditManagers = computed(() => auth.can('projectmanager:assign'));
const isRoot = computed(() => decoded.value === '/');

const USER_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/;
const GROUP_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

/** `*` and `@group` are editable; `$authenticated`, `$anonymous` and `&alias` are not. */
function principalKind(principal: string): 'user' | 'group' | 'global' | 'special' {
  if (principal === '*') {
    return 'global';
  }

  if (principal.startsWith('@')) {
    return 'group';
  }

  if (principal.startsWith('$') || principal.startsWith('&')) {
    return 'special';
  }

  return 'user';
}

function isEditable(principal: string): boolean {
  return principalKind(principal) !== 'special';
}

function kindLabel(principal: string): string {
  switch (principalKind(principal)) {
    case 'group':
      return t('accessPathDetail.kindGroup');
    case 'global':
      return t('accessPathDetail.kindGlobal');
    case 'special':
      return t('accessPathDetail.kindSpecial');
    default:
      return t('accessPathDetail.kindUser');
  }
}

function actionErrorText(cause: unknown): string {
  if (cause instanceof ApiError) {
    if (cause.status === 403) {
      return t('accessPathDetail.forbiddenPath');
    }

    if (cause.status === 409) {
      return `${t('common.conflict')}: ${t('common.conflictHint')}`;
    }
  }

  const described = describeError(toError(cause));

  return `${described.title}: ${described.detail}`;
}

async function load(): Promise<void> {
  if (decoded.value === null) {
    accessPath.value = null;
    forbidden.value = false;
    error.value = new Error(t('accessPathDetail.invalidPath'));
    loading.value = false;

    return;
  }

  loading.value = true;
  error.value = null;
  forbidden.value = false;

  try {
    accessPath.value = await getAccessPath(props.path);
  } catch (cause) {
    accessPath.value = null;

    if (cause instanceof ApiError && cause.status === 403) {
      forbidden.value = true;
    } else {
      error.value = toError(cause);
    }
  } finally {
    loading.value = false;
  }
}

/** Re-reads the ACL without flipping the whole page into the loading state. */
async function reloadAcl(): Promise<void> {
  try {
    accessPath.value = await getAccessPath(props.path);
  } catch (cause) {
    error.value = toError(cause);
  }
}

// -- ACL edits -------------------------------------------------------------

const savingPrincipals = ref<string[]>([]);
const aclError = ref('');

function isSaving(principal: string): boolean {
  return savingPrincipals.value.includes(principal);
}

async function changePermission(entry: ACLEntry, event: Event): Promise<void> {
  const next = (event.target as HTMLSelectElement).value as Permission;

  if (next === entry.permission || isSaving(entry.principal)) {
    return;
  }

  aclError.value = '';
  savingPrincipals.value = [...savingPrincipals.value, entry.principal];

  try {
    accessPath.value = await setAccessPathPermission(props.path, entry.principal, next);
    toast.push(t('accessPathDetail.permissionSaved'));
  } catch (cause) {
    aclError.value = actionErrorText(cause);
    // Put the select back on the stored value.
    await reloadAcl();
  } finally {
    savingPrincipals.value = savingPrincipals.value.filter((item) => item !== entry.principal);
  }
}

async function removeEntry(entry: ACLEntry): Promise<void> {
  if (isSaving(entry.principal)) {
    return;
  }

  aclError.value = '';
  savingPrincipals.value = [...savingPrincipals.value, entry.principal];

  try {
    await deleteAccessPathPermission(props.path, entry.principal);
    await reloadAcl();
    toast.push(t('accessPathDetail.removed'));
  } catch (cause) {
    aclError.value = actionErrorText(cause);
  } finally {
    savingPrincipals.value = savingPrincipals.value.filter((item) => item !== entry.principal);
  }
}

// -- add principal ---------------------------------------------------------

const addOpen = ref(false);
const addType = ref<'user' | 'group' | 'all'>('user');
const addUser = ref('');
const addGroup = ref('');
const addName = ref('');
const addPermission = ref<Permission>('rw');
const addError = ref('');
const addLoading = ref(false);
const addSaving = ref(false);
const userOptions = ref<string[]>([]);
const groupOptions = ref<string[]>([]);
const userOptionsFailed = ref(false);
const groupOptionsFailed = ref(false);

/** The dropdown is only usable when the list loaded and has entries. */
const useUserSelect = computed(() => !userOptionsFailed.value && userOptions.value.length > 0);
const useGroupSelect = computed(() => !groupOptionsFailed.value && groupOptions.value.length > 0);

const addPrincipal = computed<string>(() => {
  if (addType.value === 'all') {
    return '*';
  }

  if (addType.value === 'group') {
    const name = useGroupSelect.value ? addGroup.value : addName.value.trim();

    return name === '' ? '' : `@${name}`;
  }

  return useUserSelect.value ? addUser.value : addName.value.trim();
});

async function openAdd(): Promise<void> {
  addError.value = '';
  addType.value = 'user';
  addName.value = '';
  addPermission.value = 'rw';
  addOpen.value = true;
  addLoading.value = true;

  const [users, groups] = await Promise.allSettled([listUsers(1, 500), listGroups(1, 500)]);

  if (users.status === 'fulfilled') {
    userOptions.value = (users.value.items ?? []).map((user) => user.name);
    userOptionsFailed.value = false;
    addUser.value = userOptions.value[0] ?? '';
  } else {
    userOptions.value = [];
    userOptionsFailed.value = true;
    addUser.value = '';
  }

  if (groups.status === 'fulfilled') {
    groupOptions.value = (groups.value.items ?? []).map((group) => group.name);
    groupOptionsFailed.value = false;
    addGroup.value = groupOptions.value[0] ?? '';
  } else {
    groupOptions.value = [];
    groupOptionsFailed.value = true;
    addGroup.value = '';
  }

  addLoading.value = false;
}

function closeAdd(): void {
  if (!addSaving.value) {
    addOpen.value = false;
  }
}

function validateAdd(): string {
  const principal = addPrincipal.value;

  if (principal === '') {
    return t('accessPathDetail.principalRequired');
  }

  if (principal === '*') {
    return '';
  }

  if (principal.startsWith('@')) {
    if (!GROUP_PATTERN.test(principal.slice(1))) {
      return t('accessPathDetail.principalInvalid');
    }

    return '';
  }

  if (!USER_PATTERN.test(principal)) {
    return t('accessPathDetail.principalInvalid');
  }

  return '';
}

async function submitAdd(): Promise<void> {
  if (addSaving.value) {
    return;
  }

  addError.value = validateAdd();
  if (addError.value !== '') {
    return;
  }

  addSaving.value = true;

  try {
    accessPath.value = await setAccessPathPermission(props.path, addPrincipal.value, addPermission.value);
    addOpen.value = false;
    toast.push(t('accessPathDetail.added'));
  } catch (cause) {
    addError.value = actionErrorText(cause);
  } finally {
    addSaving.value = false;
  }
}

// -- project managers ------------------------------------------------------

const managersOpen = ref(false);
const managerOptions = ref<string[]>([]);
const selectedManagers = ref<string[]>([]);
const managerManual = ref('');
const managersLoading = ref(false);
const managersSaving = ref(false);
const managersError = ref('');
const managerOptionsFailed = ref(false);

function openManagers(): void {
  const current = accessPath.value;

  if (current === null || isRoot.value) {
    return;
  }

  managersError.value = '';
  managerManual.value = '';
  selectedManagers.value = [...current.managers];
  managerOptions.value = [...current.managers];
  managerOptionsFailed.value = false;
  managersOpen.value = true;
  managersLoading.value = true;

  void listUsers(1, 500)
    .then((list) => {
      const known = new Set((list.items ?? []).map((user) => user.name));
      const extra = current.managers.filter((manager) => !known.has(manager));

      managerOptions.value = [...extra, ...known];
    })
    .catch(() => {
      // A Project-Manager may lack user:view; editing by name still works.
      managerOptionsFailed.value = true;
    })
    .finally(() => {
      managersLoading.value = false;
    });
}

function closeManagers(): void {
  if (!managersSaving.value) {
    managersOpen.value = false;
  }
}

function addManagerByName(): void {
  const name = managerManual.value.trim();

  if (!USER_PATTERN.test(name)) {
    managersError.value = t('accessPathDetail.managerInvalid');

    return;
  }

  if (!selectedManagers.value.includes(name)) {
    selectedManagers.value = [...selectedManagers.value, name];
  }

  if (!managerOptions.value.includes(name)) {
    managerOptions.value = [...managerOptions.value, name];
  }

  managerManual.value = '';
  managersError.value = '';
}

async function submitManagers(): Promise<void> {
  if (managersSaving.value) {
    return;
  }

  managersSaving.value = true;
  managersError.value = '';

  try {
    const updated = await setAccessPathManagers(props.path, selectedManagers.value);

    if (accessPath.value !== null) {
      accessPath.value = { ...accessPath.value, managers: updated.users ?? [] };
    }

    managersOpen.value = false;
    toast.push(t('accessPathDetail.managersSaved'));
  } catch (cause) {
    managersError.value = actionErrorText(cause);
  } finally {
    managersSaving.value = false;
  }
}

// -- delete path -----------------------------------------------------------

const deleteOpen = ref(false);
const deleteConfirm = ref('');
const deleteError = ref('');
const deleteSaving = ref(false);

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
  const current = accessPath.value;

  if (current === null || deleteSaving.value) {
    return;
  }

  // Typing the exact path is the confirmation gate (plan D7 pattern).
  if (deleteConfirm.value.trim() !== current.path) {
    return;
  }

  deleteSaving.value = true;
  deleteError.value = '';

  try {
    await deleteAccessPath(props.path);
    deleteOpen.value = false;
    toast.push(t('accessPathDetail.deleteSuccess'));
    await router.push({ name: 'access-paths' });
  } catch (cause) {
    if (cause instanceof ApiError && cause.status === 403) {
      deleteError.value = t('accessPathDetail.forbiddenPath');
    } else {
      deleteError.value = actionErrorText(cause);
    }
  } finally {
    deleteSaving.value = false;
  }
}

watch(
  () => props.path,
  () => {
    addOpen.value = false;
    managersOpen.value = false;
    deleteOpen.value = false;
    aclError.value = '';
    void load();
  },
  { immediate: true },
);
</script>

<template>
  <section class="space-y-6">
    <RouterLink :to="{ name: 'access-paths' }" class="text-sm text-sky-700 hover:underline dark:text-sky-400">
      ← {{ t('accessPaths.title') }}
    </RouterLink>

    <div
      v-if="forbidden"
      class="rounded border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200"
      role="alert"
    >
      <p class="font-medium">{{ t('common.forbidden') }}</p>
      <p class="mt-1 break-words">{{ t('accessPathDetail.forbiddenPath') }}</p>
    </div>

    <StateBlock
      v-else
      :loading="loading"
      :error="error"
      :empty="accessPath === null"
      :empty-text="t('accessPathDetail.empty')"
    >
      <template v-if="accessPath">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 class="font-mono text-lg font-semibold">{{ accessPath.path }}</h1>
            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('accessPathDetail.aclNote') }}</p>
          </div>
          <button
            v-if="canDelete"
            type="button"
            class="rounded border border-red-300 px-3 py-1 text-sm text-red-700 hover:bg-red-50 dark:border-red-900 dark:text-red-300 dark:hover:bg-red-950"
            @click="openDelete"
          >
            {{ t('accessPathDetail.delete') }}
          </button>
        </div>

        <div class="mt-4 overflow-hidden rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
          <div class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-4 py-3 dark:border-slate-800">
            <h2 class="text-sm font-semibold">{{ t('accessPathDetail.acl') }} ({{ accessPath.acl.length }})</h2>
            <button
              v-if="canAssign"
              type="button"
              class="text-xs text-sky-700 hover:underline dark:text-sky-400"
              @click="openAdd"
            >
              {{ t('accessPathDetail.addPrincipal') }}
            </button>
          </div>

          <p
            v-if="aclError"
            class="mx-4 mt-3 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
            role="alert"
          >
            {{ aclError }}
          </p>

          <p v-if="accessPath.acl.length === 0" class="px-4 py-6 text-center text-sm text-slate-500 dark:text-slate-400">
            {{ t('accessPathDetail.emptyAcl') }}
          </p>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-left text-sm">
              <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500 dark:bg-slate-800/60 dark:text-slate-400">
                <tr>
                  <th class="px-4 py-2 font-medium">{{ t('accessPaths.principal') }}</th>
                  <th class="px-4 py-2 font-medium">{{ t('accessPaths.permission') }}</th>
                  <th class="px-4 py-2 font-medium">{{ t('accessPathDetail.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in accessPath.acl" :key="entry.principal" class="border-t border-slate-100 dark:border-slate-800">
                  <td class="px-4 py-2">
                    <span class="font-mono text-xs">{{ entry.principal }}</span>
                    <span class="ml-2 rounded bg-slate-100 px-1.5 py-0.5 text-[11px] text-slate-600 dark:bg-slate-800 dark:text-slate-300">
                      {{ kindLabel(entry.principal) }}
                    </span>
                  </td>
                  <td class="px-4 py-2">
                    <select
                      v-if="isEditable(entry.principal) && canAssign"
                      :value="entry.permission"
                      :disabled="isSaving(entry.principal)"
                      :aria-label="t('accessPaths.permission')"
                      class="rounded border border-slate-300 bg-white px-2 py-1 text-xs dark:border-slate-700 dark:bg-slate-800"
                      @change="changePermission(entry, $event)"
                    >
                      <option value="r">r</option>
                      <option value="rw">rw</option>
                      <option value="none">none</option>
                    </select>
                    <span v-else class="font-mono text-xs">{{ entry.permission }}</span>
                  </td>
                  <td class="px-4 py-2">
                    <button
                      v-if="isEditable(entry.principal) && canUnassign"
                      type="button"
                      class="text-xs text-red-700 hover:underline disabled:opacity-50 dark:text-red-300"
                      :disabled="isSaving(entry.principal)"
                      @click="removeEntry(entry)"
                    >
                      {{ isSaving(entry.principal) ? t('accessPathDetail.removing') : t('accessPathDetail.remove') }}
                    </button>
                    <span v-else class="text-xs text-slate-400 dark:text-slate-500">{{ t('accessPathDetail.readOnlyPrincipal') }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="mt-4 rounded border border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
          <div class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-4 py-3 dark:border-slate-800">
            <h2 class="text-sm font-semibold">{{ t('accessPaths.managers') }} ({{ accessPath.managers.length }})</h2>
            <button
              v-if="canEditManagers"
              type="button"
              class="text-xs text-sky-700 hover:underline disabled:cursor-not-allowed disabled:text-slate-400 disabled:no-underline dark:text-sky-400 dark:disabled:text-slate-500"
              :disabled="isRoot"
              @click="openManagers"
            >
              {{ t('accessPathDetail.editManagers') }}
            </button>
          </div>
          <p v-if="isRoot" class="px-4 pt-3 text-xs text-amber-700 dark:text-amber-300">
            {{ t('accessPathDetail.rootManagers') }}
          </p>
          <p v-if="accessPath.managers.length === 0" class="px-4 py-6 text-center text-sm text-slate-500 dark:text-slate-400">
            {{ t('accessPathDetail.noManagers') }}
          </p>
          <ul v-else class="flex flex-wrap gap-2 px-4 py-3">
            <li
              v-for="manager in accessPath.managers"
              :key="manager"
              class="rounded bg-amber-100 px-2 py-1 text-xs text-amber-900 dark:bg-amber-900/40 dark:text-amber-200"
            >
              {{ manager }}
            </li>
          </ul>
        </div>

        <ModalDialog :open="addOpen" :title="t('accessPathDetail.addPrincipalTitle')" @close="closeAdd">
          <form id="add-principal-form" class="space-y-4" @submit.prevent="submitAdd">
            <div>
              <label for="principal-type" class="block text-sm font-medium">{{ t('accessPathDetail.principalType') }}</label>
              <select
                id="principal-type"
                v-model="addType"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              >
                <option value="user">{{ t('accessPathDetail.kindUser') }}</option>
                <option value="group">{{ t('accessPathDetail.kindGroup') }}</option>
                <option value="all">{{ t('accessPathDetail.kindGlobal') }}</option>
              </select>
            </div>

            <p v-if="addLoading" class="text-sm text-slate-500 dark:text-slate-400">{{ t('common.loading') }}</p>

            <div v-else-if="addType === 'user'">
              <label for="principal-user" class="block text-sm font-medium">{{ t('accessPathDetail.principalName') }}</label>
              <select
                v-if="useUserSelect"
                id="principal-user"
                v-model="addUser"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              >
                <option v-for="name in userOptions" :key="name" :value="name">{{ name }}</option>
              </select>
              <input
                v-else
                id="principal-user"
                v-model="addName"
                type="text"
                autocomplete="off"
                spellcheck="false"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
              />
              <p v-if="userOptionsFailed" class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {{ t('accessPathDetail.optionsUnavailable') }}
              </p>
            </div>

            <div v-else-if="addType === 'group'">
              <label for="principal-group" class="block text-sm font-medium">{{ t('accessPathDetail.principalName') }}</label>
              <select
                v-if="useGroupSelect"
                id="principal-group"
                v-model="addGroup"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              >
                <option v-for="name in groupOptions" :key="name" :value="name">@{{ name }}</option>
              </select>
              <input
                v-else
                id="principal-group"
                v-model="addName"
                type="text"
                autocomplete="off"
                spellcheck="false"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
              />
              <p v-if="groupOptionsFailed" class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {{ t('accessPathDetail.optionsUnavailable') }}
              </p>
            </div>

            <p v-else class="rounded border border-slate-200 px-3 py-2 font-mono text-sm dark:border-slate-800">*</p>

            <div>
              <label for="principal-permission" class="block text-sm font-medium">{{ t('accessPaths.permission') }}</label>
              <select
                id="principal-permission"
                v-model="addPermission"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              >
                <option value="r">r</option>
                <option value="rw">rw</option>
                <option value="none">none</option>
              </select>
              <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t('accessPathDetail.permissionHint') }}</p>
            </div>

            <p
              v-if="addError"
              class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
              role="alert"
            >
              {{ addError }}
            </p>
          </form>

          <template #footer>
            <div class="flex justify-end gap-2">
              <button
                type="button"
                class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
                :disabled="addSaving"
                @click="closeAdd"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="submit"
                form="add-principal-form"
                class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
                :disabled="addSaving || addLoading"
              >
                {{ addSaving ? t('accessPathDetail.adding') : t('common.save') }}
              </button>
            </div>
          </template>
        </ModalDialog>

        <ModalDialog :open="managersOpen" :title="t('accessPathDetail.editManagers')" @close="closeManagers">
          <form id="edit-managers-form" class="space-y-3" @submit.prevent="submitManagers">
            <p class="text-xs text-slate-500 dark:text-slate-400">{{ t('accessPathDetail.managersHint') }}</p>

            <p v-if="managersLoading" class="text-sm text-slate-500 dark:text-slate-400">{{ t('common.loading') }}</p>
            <div v-else class="max-h-64 space-y-1 overflow-y-auto">
              <label
                v-for="name in managerOptions"
                :key="name"
                class="flex items-center gap-2 rounded border border-slate-200 px-3 py-2 text-sm dark:border-slate-800"
              >
                <input v-model="selectedManagers" type="checkbox" :value="name" />
                <span class="font-mono">{{ name }}</span>
              </label>
            </div>

            <div v-if="managerOptionsFailed" class="space-y-2">
              <p class="text-xs text-slate-500 dark:text-slate-400">{{ t('accessPathDetail.managersUsersUnavailable') }}</p>
              <div class="flex gap-2">
                <input
                  v-model="managerManual"
                  type="text"
                  autocomplete="off"
                  spellcheck="false"
                  :placeholder="t('accessPathDetail.managerName')"
                  :aria-label="t('accessPathDetail.managerName')"
                  class="w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
                  @keydown.enter.prevent="addManagerByName"
                />
                <button
                  type="button"
                  class="shrink-0 rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
                  @click="addManagerByName"
                >
                  {{ t('accessPathDetail.managerAdd') }}
                </button>
              </div>
            </div>

            <p
              v-if="managersError"
              class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
              role="alert"
            >
              {{ managersError }}
            </p>
          </form>

          <template #footer>
            <div class="flex justify-end gap-2">
              <button
                type="button"
                class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
                :disabled="managersSaving"
                @click="closeManagers"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="submit"
                form="edit-managers-form"
                class="rounded bg-sky-600 px-3 py-1 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
                :disabled="managersSaving || managersLoading"
              >
                {{ managersSaving ? t('userDetail.saving') : t('common.save') }}
              </button>
            </div>
          </template>
        </ModalDialog>

        <ModalDialog :open="deleteOpen" :title="t('accessPathDetail.deleteTitle')" @close="closeDelete">
          <form id="delete-access-path-form" class="space-y-4" @submit.prevent="submitDelete">
            <p class="text-sm text-red-700 dark:text-red-300">{{ t('accessPathDetail.deleteHint') }}</p>

            <p class="text-sm">
              {{ t('accessPathDetail.deleteConfirmLabel') }}
              <span class="break-all font-mono font-semibold">{{ accessPath.path }}</span>
            </p>

            <div>
              <label for="delete-access-path-confirm" class="block text-sm font-medium">
                {{ t('accessPathDetail.deleteConfirmField') }}
              </label>
              <input
                id="delete-access-path-confirm"
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
                form="delete-access-path-form"
                class="rounded bg-red-600 px-3 py-1 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
                :disabled="deleteSaving || deleteConfirm.trim() !== accessPath.path"
              >
                {{ deleteSaving ? t('accessPathDetail.deleting') : t('accessPathDetail.deleteSubmit') }}
              </button>
            </div>
          </template>
        </ModalDialog>
      </template>
    </StateBlock>
  </section>
</template>
