<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import {
  ApiError,
  deleteGroup,
  getGroup,
  listUsers,
  setGroupMembers,
} from '../api/client';
import { encodeAccessPath } from '../api/encoding';
import { describeError, toError } from '../api/errors';
import type { GroupDetail, User } from '../api/types';
import ModalDialog from '../components/ModalDialog.vue';
import StateBlock from '../components/StateBlock.vue';
import { t } from '../i18n';
import { useToastStore } from '../stores/toast';

const props = defineProps<{ name: string }>();

const toast = useToastStore();
const router = useRouter();

const group = ref<GroupDetail | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);

const canEditMembers = computed(() => true);
const canDelete = computed(() => true);
const accessPaths = computed(() => group.value?.accessPaths ?? []);

// -- members ---------------------------------------------------------------

const membersOpen = ref(false);
const memberOptions = ref<User[]>([]);
const selectedMembers = ref<string[]>([]);
const membersLoading = ref(false);
const membersSaving = ref(false);
const membersError = ref('');

// -- delete ----------------------------------------------------------------

const deleteOpen = ref(false);
const deleteConfirm = ref('');
const deleteError = ref('');
const deleteSaving = ref(false);

/** One-line rendering of any API failure, including `operation_failed` steps. */
function actionErrorText(cause: unknown): string {
  const described = describeError(toError(cause));

  return `${described.title}: ${described.detail}`;
}

async function load(): Promise<void> {
  loading.value = true;
  error.value = null;

  try {
    group.value = await getGroup(props.name);
  } catch (cause) {
    group.value = null;
    error.value = toError(cause);
  } finally {
    loading.value = false;
  }
}

async function openMembers(): Promise<void> {
  const current = group.value;

  if (current === null) {
    return;
  }

  membersError.value = '';
  selectedMembers.value = [...current.members];
  membersOpen.value = true;
  membersLoading.value = true;

  try {
    const list = await listUsers(1, 500);
    const known = new Set((list.items ?? []).map((user) => user.name));

    // Keep members that are not in svn-auth-file visible, otherwise saving
    // would silently drop them.
    const extra = current.members.filter((member) => !known.has(member));

    memberOptions.value = [
      ...extra.map((name) => ({ name, groups: [] }) as User),
      ...(list.items ?? []),
    ];
  } catch (cause) {
    memberOptions.value = current.members.map((name) => ({ name, groups: [] }) as User);
    membersError.value = actionErrorText(cause);
  } finally {
    membersLoading.value = false;
  }
}

function closeMembers(): void {
  if (!membersSaving.value) {
    membersOpen.value = false;
  }
}

async function submitMembers(): Promise<void> {
  const current = group.value;

  if (current === null || membersSaving.value) {
    return;
  }

  membersSaving.value = true;
  membersError.value = '';

  try {
    const updated = await setGroupMembers(current.name, selectedMembers.value);
    group.value = { ...current, members: updated.users ?? [] };
    membersOpen.value = false;
    toast.push(t('groupDetail.membersSaved'));
  } catch (cause) {
    membersError.value = actionErrorText(cause);
  } finally {
    membersSaving.value = false;
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
  const current = group.value;

  if (current === null || deleteSaving.value) {
    return;
  }

  // Typing the exact group name is the confirmation gate (plan D7 pattern).
  if (deleteConfirm.value.trim() !== current.name) {
    return;
  }

  deleteSaving.value = true;
  deleteError.value = '';

  try {
    await deleteGroup(current.name);
    deleteOpen.value = false;
    toast.push(t('groupDetail.deleteSuccess'));
    await router.push({ name: 'groups' });
  } catch (cause) {
    if (cause instanceof ApiError && cause.status === 403) {
      deleteError.value = t('common.forbiddenHint');
    } else {
      deleteError.value = actionErrorText(cause);
    }
  } finally {
    deleteSaving.value = false;
  }
}

watch(
  () => props.name,
  () => {
    membersOpen.value = false;
    deleteOpen.value = false;
    void load();
  },
  { immediate: true },
);
</script>

<template>
  <section class="space-y-6">
    <RouterLink :to="{ name: 'groups' }" class="text-sm text-[#0969da] hover:underline dark:text-[#2f81f7]">
      ← {{ t('groups.title') }}
    </RouterLink>

    <StateBlock :loading="loading" :error="error" :empty="group === null" :empty-text="t('groupDetail.empty')">
      <template v-if="group">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 class="text-xl font-semibold">{{ group.name }}</h1>
            <p v-if="group.description" class="mt-1 text-sm text-[#656d76] dark:text-[#c9d1d9]">{{ group.description }}</p>
            <p class="mt-1 text-xs text-[#656d76] dark:text-[#8b949e]">{{ t('groupDetail.aclSyncNote') }}</p>
          </div>
          <button
            v-if="canDelete"
            type="button"
            class="rounded-md border border-red-300 px-3 py-1 text-sm text-red-700 hover:bg-red-50 dark:border-red-900 dark:text-red-300 dark:hover:bg-red-950"
            @click="openDelete"
          >
            {{ t('groupDetail.delete') }}
          </button>
        </div>

        <div class="mt-4 rounded-md border border-[#d0d7de] bg-[#ffffff] dark:border-[#30363d] dark:bg-[#161b22]">
          <div class="flex items-center justify-between gap-2 border-b border-[#d0d7de] px-4 py-3 dark:border-[#30363d]">
            <h2 class="text-sm font-semibold">
              {{ t('groupDetail.members') }} ({{ group.members.length }})
            </h2>
            <button
              v-if="canEditMembers"
              type="button"
              class="text-xs text-[#0969da] hover:underline dark:text-[#2f81f7]"
              @click="openMembers"
            >
              {{ t('groupDetail.editMembers') }}
            </button>
          </div>
          <p v-if="group.members.length === 0" class="px-4 py-6 text-center text-sm text-[#656d76] dark:text-[#8b949e]">
            {{ t('groupDetail.emptyMembers') }}
          </p>
          <ul v-else class="flex flex-wrap gap-2 px-4 py-3">
            <li
              v-for="member in group.members"
              :key="member"
              class="rounded-md bg-[#f6f8fa] px-2 py-1 text-xs font-medium dark:bg-[#21262d]"
            >
              <RouterLink :to="{ name: 'user-detail', params: { name: member } }" class="hover:underline">
                {{ member }}
              </RouterLink>
            </li>
          </ul>
        </div>

        <div class="mt-4 rounded-md border border-[#d0d7de] bg-[#ffffff] dark:border-[#30363d] dark:bg-[#161b22]">
          <h2 class="border-b border-[#d0d7de] px-4 py-3 text-sm font-semibold dark:border-[#30363d]">
            {{ t('groupDetail.accessPaths') }} ({{ accessPaths.length }})
          </h2>
          <p
            v-if="accessPaths.length === 0"
            class="px-4 py-6 text-center text-sm text-[#656d76] dark:text-[#8b949e]"
          >
            {{ t('groupDetail.emptyAccessPaths') }}
          </p>
          <ul v-else class="divide-y divide-[#eaeef2] dark:divide-[#30363d]">
            <li v-for="path in accessPaths" :key="path" class="px-4 py-2 font-mono text-xs">
              <RouterLink
                :to="{ name: 'access-path-detail', params: { path: encodeAccessPath(path) } }"
                class="text-[#0969da] hover:underline dark:text-[#2f81f7]"
              >
                {{ path }}
              </RouterLink>
            </li>
          </ul>
        </div>

        <ModalDialog :open="membersOpen" :title="t('groupDetail.membersTitle')" @close="closeMembers">
          <form id="edit-members-form" class="space-y-3" @submit.prevent="submitMembers">
            <p class="text-xs text-[#656d76] dark:text-[#8b949e]">{{ t('groupDetail.membersHint') }}</p>

            <p v-if="membersLoading" class="text-sm text-[#656d76] dark:text-[#8b949e]">{{ t('common.loading') }}</p>
            <p v-else-if="memberOptions.length === 0" class="text-sm text-[#656d76] dark:text-[#8b949e]">
              {{ t('groupDetail.noUsers') }}
            </p>
            <div v-else class="max-h-72 space-y-1 overflow-y-auto">
              <label
                v-for="user in memberOptions"
                :key="user.name"
                class="flex items-center gap-2 rounded-md border border-[#d0d7de] px-3 py-2 text-sm dark:border-[#30363d]"
              >
                <input v-model="selectedMembers" type="checkbox" :value="user.name" />
                <span class="font-mono">{{ user.name }}</span>
              </label>
            </div>

            <p
              v-if="membersError"
              class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
              role="alert"
            >
              {{ membersError }}
            </p>
          </form>

          <template #footer>
            <div class="flex justify-end gap-2">
              <button
                type="button"
                class="rounded-md border border-[#d0d7de] px-3 py-1 text-sm hover:bg-[#f6f8fa] disabled:opacity-50 dark:border-[#30363d] dark:hover:bg-[#21262d]"
                :disabled="membersSaving"
                @click="closeMembers"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="submit"
                form="edit-members-form"
                class="rounded-md bg-[#1f883d] px-3 py-1 text-sm font-medium text-white hover:bg-[#1a7f37] disabled:opacity-50"
                :disabled="membersSaving || membersLoading"
              >
                {{ membersSaving ? t('userDetail.saving') : t('common.save') }}
              </button>
            </div>
          </template>
        </ModalDialog>

        <ModalDialog :open="deleteOpen" :title="t('groupDetail.deleteTitle')" @close="closeDelete">
          <form id="delete-group-form" class="space-y-4" @submit.prevent="submitDelete">
            <p class="text-sm text-red-700 dark:text-red-300">{{ t('groupDetail.deleteHint') }}</p>

            <p class="text-sm">
              {{ t('groupDetail.deleteConfirmLabel') }}
              <span class="font-mono font-semibold">{{ group.name }}</span>
            </p>

            <div>
              <label for="delete-group-confirm" class="block text-sm font-medium">
                {{ t('groupDetail.deleteConfirmField') }}
              </label>
              <input
                id="delete-group-confirm"
                v-model="deleteConfirm"
                type="text"
                autocomplete="off"
                spellcheck="false"
                class="mt-1 w-full rounded-md border border-[#d0d7de] bg-[#ffffff] px-3 py-2 font-mono text-sm dark:border-[#30363d] dark:bg-[#21262d]"
              />
            </div>

            <p
              v-if="deleteError"
              class="rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
              role="alert"
            >
              {{ deleteError }}
            </p>
          </form>

          <template #footer>
            <div class="flex justify-end gap-2">
              <button
                type="button"
                class="rounded-md border border-[#d0d7de] px-3 py-1 text-sm hover:bg-[#f6f8fa] disabled:opacity-50 dark:border-[#30363d] dark:hover:bg-[#21262d]"
                :disabled="deleteSaving"
                @click="closeDelete"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                type="submit"
                form="delete-group-form"
                class="rounded-md bg-red-600 px-3 py-1 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
                :disabled="deleteSaving || deleteConfirm.trim() !== group.name"
              >
                {{ deleteSaving ? t('groupDetail.deleting') : t('groupDetail.deleteSubmit') }}
              </button>
            </div>
          </template>
        </ModalDialog>
      </template>
    </StateBlock>
  </section>
</template>
