<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import { ApiError, createAdmin, getSetupStatus } from '../api/client';
import type { SetupResult, SetupStatus } from '../api/types';
import { t } from '../i18n';

const router = useRouter();

/** Sentinel value of the "create a new account…" option. */
const NEW_ACCOUNT = '__new_account__';

/** Mirrors the API's account-name rule (same as the Users create form). */
const USERNAME_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/;
const MAX_PASSWORD_LENGTH = 256;

const status = ref<SetupStatus | null>(null);
const loading = ref(true);
const loadError = ref('');

const selectedAccount = ref('');
const newUsername = ref('');
const password = ref('');
const confirmPassword = ref('');
const bootstrapToken = ref('');
const submitting = ref(false);
const formError = ref('');
const result = ref<SetupResult | null>(null);

const users = computed<string[]>(() => status.value?.users ?? []);
const hasUsers = computed(() => users.value.length > 0);
const canCreateAccount = computed(() => status.value?.canCreateAccount ?? false);
const setupRequired = computed(() => status.value?.setupRequired ?? true);

/** True when the account picker is shown instead of a free-text field. */
const usesSelect = computed(() => hasUsers.value);

/** True when the chosen account does not exist yet, so a password is needed. */
const creatingAccount = computed(() => {
  if (!usesSelect.value) {
    return true;
  }

  if (selectedAccount.value === NEW_ACCOUNT) {
    return true;
  }

  return !users.value.includes(selectedAccount.value);
});

/** The account name that will be sent to the API. */
const accountName = computed(() => {
  if (usesSelect.value && selectedAccount.value !== NEW_ACCOUNT) {
    return selectedAccount.value;
  }

  return newUsername.value.trim();
});

interface Diagnostic {
  tone: 'error' | 'warning' | 'info';
  text: string;
}

/** File/directory problems the operator can fix before submitting. */
const diagnostics = computed<Diagnostic[]>(() => {
  const current = status.value;
  if (current === null) {
    return [];
  }

  const items: Diagnostic[] = [];
  const authFile = current.authFile;
  const accessFile = current.accessFile;

  if (!authFile.exists) {
    items.push({ tone: 'error', text: t('setup.authFileMissing') });
  } else if (!authFile.writable) {
    items.push({ tone: 'error', text: t('setup.authFileUnwritable') });
  }

  if (!accessFile.exists) {
    items.push({ tone: 'error', text: t('setup.accessFileMissing') });
  } else if (!accessFile.writable) {
    items.push({ tone: 'error', text: t('setup.accessFileUnwritable') });
  }

  if (!current.roleFile.exists || !current.roleFile.administratorPresent) {
    items.push({ tone: 'info', text: t('setup.roleFileWillCreate') });
  }

  if (!current.canCreateAccount) {
    items.push({ tone: 'error', text: t('setup.cannotCreateAccount') });
  }

  return items;
});

const diagnosticClass: Record<Diagnostic['tone'], string> = {
  error:
    'border-red-300 bg-red-50 text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200',
  warning:
    'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200',
  info: 'border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300',
};

/** Human readable names for the files reported in `initializedFiles`. */
function fileLabel(name: string): string {
  switch (name) {
    case 'authFile':
      return t('setup.fileAuth');
    case 'accessFile':
      return t('setup.fileAccess');
    case 'roleFile':
      return t('setup.fileRole');
    default:
      return name;
  }
}

const initializedFiles = computed<string[]>(() =>
  (result.value?.initializedFiles ?? []).map((name) => fileLabel(name)),
);

/** Server reason for the pending setup (shown verbatim in the diagnostics). */
const statusMessage = computed(() => (status.value === null ? '' : status.value.reason));

/** Loads (or reloads) GET /setup/status. */
async function loadStatus(): Promise<void> {
  loading.value = true;
  loadError.value = '';

  try {
    const next = await getSetupStatus();
    status.value = next;

    if (next.users.length > 0) {
      selectedAccount.value = next.users[0];
    } else {
      selectedAccount.value = '';
    }
  } catch (cause) {
    status.value = null;
    loadError.value = cause instanceof ApiError ? cause.message : t('setup.statusLoadFailed');
  } finally {
    loading.value = false;
  }
}

/** Client-side checks; the API repeats them and stays authoritative. */
function validate(): string {
  if (bootstrapToken.value === '') {
    return t('setup.tokenRequired');
  }

  if (accountName.value === '') {
    return t('setup.newUsernameRequired');
  }

  if (creatingAccount.value && !USERNAME_PATTERN.test(accountName.value)) {
    return t('setup.newUsernameInvalid');
  }

  if (!creatingAccount.value) {
    return '';
  }

  if (password.value === '') {
    return t('setup.passwordRequired');
  }

  if (password.value.length > MAX_PASSWORD_LENGTH) {
    return t('setup.passwordTooLong');
  }

  if (/[\r\n]/.test(password.value)) {
    return t('setup.passwordNewline');
  }

  if (password.value !== confirmPassword.value) {
    return t('setup.passwordMismatch');
  }

  return '';
}

/** Prefers the server's own message over the generic envelope text. */
function serverMessage(cause: ApiError, fallback: string): string {
  const message = cause.body?.message ?? cause.message;

  return message === '' ? fallback : message;
}

/** The field named by the server, when it sends structured details. */
function serverField(cause: ApiError): string {
  const field = cause.body?.details?.field;

  return typeof field === 'string' ? field : '';
}

function errorMessage(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('setup.networkError');
  }

  const detail = serverMessage(cause, '');

  switch (cause.status) {
    case 0:
      return t('setup.networkError');
    case 403:
      return t('setup.invalidToken');
    case 409:
      return t('setup.alreadySetUp');
    case 422: {
      const field = serverField(cause);
      // The API only names the field in the message today; accept both.
      if (field === 'password' || (field === '' && /password/i.test(detail))) {
        return detail === '' ? t('setup.passwordRequired') : detail;
      }

      return detail === '' ? t('setup.failed') : detail;
    }
    case 503:
      if (cause.code === 'storage_unavailable') {
        return detail === '' ? t('setup.storageUnavailable') : `${t('setup.storageUnavailable')} ${detail}`;
      }

      return `${t('setup.failed')} (${cause.status})`;
    default:
      return `${t('setup.failed')} (${cause.status})`;
  }
}

async function submit(): Promise<void> {
  if (submitting.value || !canCreateAccount.value) {
    return;
  }

  formError.value = validate();
  if (formError.value !== '') {
    return;
  }

  submitting.value = true;
  result.value = null;

  try {
    const created = await createAdmin(
      accountName.value,
      creatingAccount.value ? password.value : '',
      bootstrapToken.value,
    );
    result.value = created;
    password.value = '';
    confirmPassword.value = '';
    bootstrapToken.value = '';
  } catch (cause) {
    if (cause instanceof ApiError && cause.status === 409) {
      formError.value = t('setup.alreadySetUp');
      await loadStatus();
    } else {
      formError.value = errorMessage(cause);
    }
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  void loadStatus();
});
</script>

<template>
  <section class="mx-auto mt-10 max-w-xl px-4">
    <h1 class="text-xl font-semibold">{{ t('setup.title') }}</h1>
    <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">{{ t('setup.intro') }}</p>

    <p v-if="loading" class="mt-6 text-sm text-slate-500 dark:text-slate-400" role="status">
      {{ t('setup.loading') }}
    </p>

    <div
      v-else-if="loadError !== ''"
      class="mt-6 rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
      role="alert"
    >
      <p class="font-medium">{{ t('setup.statusLoadFailed') }}</p>
      <p class="mt-1 break-words">{{ loadError }}</p>
      <button
        type="button"
        class="mt-3 rounded border border-red-400 px-3 py-1 text-sm font-medium hover:bg-red-100 dark:border-red-700 dark:hover:bg-red-900"
        @click="loadStatus()"
      >
        {{ t('setup.retry') }}
      </button>
    </div>

    <div
      v-else-if="!setupRequired"
      class="mt-6 rounded border border-emerald-300 bg-emerald-50 px-4 py-4 text-sm text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-200"
      role="status"
    >
      <p class="font-medium">{{ t('setup.doneTitle') }}</p>
      <p class="mt-1">{{ t('setup.doneHint') }}</p>
      <button
        type="button"
        class="mt-3 rounded bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700"
        @click="router.push({ name: 'login' })"
      >
        {{ t('setup.gotoLogin') }}
      </button>
    </div>

    <div
      v-else-if="result !== null"
      class="mt-6 rounded border border-emerald-300 bg-emerald-50 px-4 py-4 text-sm text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-200"
      role="status"
    >
      <p class="font-medium">{{ t('setup.success') }}</p>
      <dl class="mt-2 space-y-1">
        <div class="flex gap-2">
          <dt class="font-medium">{{ t('setup.successAccount') }}</dt>
          <dd class="font-mono">{{ result.username }}</dd>
        </div>
        <div class="flex gap-2">
          <dt class="font-medium">{{ t('setup.successMode') }}</dt>
          <dd>
            {{ result.accountCreated ? t('setup.accountCreated') : t('setup.accountReused') }}
          </dd>
        </div>
        <div class="flex gap-2">
          <dt class="font-medium">{{ t('setup.initializedFiles') }}</dt>
          <dd>
            <span v-if="initializedFiles.length === 0">{{ t('setup.noFilesInitialized') }}</span>
            <span v-else>{{ initializedFiles.join('、') }}</span>
          </dd>
        </div>
      </dl>
      <p class="mt-2">{{ t('setup.successHint') }}</p>
      <button
        type="button"
        class="mt-3 rounded bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700"
        @click="router.push({ name: 'login' })"
      >
        {{ t('setup.gotoLogin') }}
      </button>
    </div>

    <template v-else>
      <section class="mt-6 space-y-2" aria-labelledby="setup-diagnostics">
        <h2 id="setup-diagnostics" class="text-sm font-semibold">{{ t('setup.diagnostics') }}</h2>
        <p v-if="diagnostics.length === 0" class="text-sm text-slate-500 dark:text-slate-400">
          {{ t('setup.diagnosticsOk') }}
        </p>
        <ul v-else class="space-y-2">
          <li
            v-for="item in diagnostics"
            :key="item.text"
            class="rounded border px-3 py-2 text-sm"
            :class="diagnosticClass[item.tone]"
            role="status"
          >
            {{ item.text }}
          </li>
        </ul>
        <p class="text-xs text-slate-500 dark:text-slate-400">
          {{ t('setup.reasonLabel') }}: {{ statusMessage === '' ? t('common.none') : statusMessage }}
        </p>
      </section>

      <form
        class="mt-6 space-y-4 rounded border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900"
        @submit.prevent="submit"
      >
        <fieldset class="space-y-4" :disabled="!canCreateAccount || submitting">
          <legend class="sr-only">{{ t('setup.title') }}</legend>

          <div>
            <label for="setup-account" class="block text-sm font-medium">{{ t('setup.account') }}</label>

            <select
              v-if="usesSelect"
              id="setup-account"
              v-model="selectedAccount"
              name="account"
              class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
            >
              <option v-for="name in users" :key="name" :value="name">{{ name }}</option>
              <option :value="NEW_ACCOUNT">{{ t('setup.newAccountOption') }}</option>
            </select>

            <input
              v-else
              id="setup-account"
              v-model="newUsername"
              name="username"
              type="text"
              autocomplete="username"
              required
              class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
            />

            <p v-if="usesSelect && selectedAccount !== NEW_ACCOUNT" class="text-xs text-slate-500 dark:text-slate-400">
              {{ t('setup.existingAccountHint') }}
            </p>
            <p v-else class="text-xs text-slate-500 dark:text-slate-400">
              {{ t('setup.newUsernameHint') }}
            </p>
          </div>

          <div v-if="usesSelect && selectedAccount === NEW_ACCOUNT">
            <label for="setup-new-username" class="block text-sm font-medium">{{ t('setup.newUsername') }}</label>
            <input
              id="setup-new-username"
              v-model="newUsername"
              name="newUsername"
              type="text"
              autocomplete="off"
              class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
            />
          </div>

          <template v-if="creatingAccount">
            <div>
              <label for="setup-password" class="block text-sm font-medium">{{ t('setup.password') }}</label>
              <p class="text-xs text-slate-500 dark:text-slate-400">{{ t('setup.passwordHint') }}</p>
              <input
                id="setup-password"
                v-model="password"
                name="password"
                type="password"
                autocomplete="new-password"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              />
            </div>

            <div>
              <label for="setup-confirm-password" class="block text-sm font-medium">
                {{ t('setup.confirmPassword') }}
              </label>
              <input
                id="setup-confirm-password"
                v-model="confirmPassword"
                name="confirmPassword"
                type="password"
                autocomplete="new-password"
                class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
              />
            </div>
          </template>

          <p v-else class="rounded border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300">
            {{ t('setup.passwordKeepHint') }}
          </p>

          <div>
            <label for="setup-token" class="block text-sm font-medium">{{ t('setup.token') }}</label>
            <p class="text-xs text-slate-500 dark:text-slate-400">{{ t('setup.tokenHint') }}</p>
            <input
              id="setup-token"
              v-model="bootstrapToken"
              name="bootstrapToken"
              type="password"
              autocomplete="off"
              required
              class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
            />
          </div>
        </fieldset>

        <p
          v-if="formError"
          class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
          role="alert"
        >
          {{ formError }}
        </p>

        <button
          type="submit"
          class="w-full rounded bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
          :disabled="submitting || !canCreateAccount"
        >
          {{ submitting ? t('setup.submitting') : t('setup.submit') }}
        </button>

        <p class="text-center text-xs text-slate-500 dark:text-slate-400">
          <RouterLink to="/login" class="text-sky-600 hover:underline dark:text-sky-400">
            {{ t('setup.backToLogin') }}
          </RouterLink>
        </p>
      </form>
    </template>
  </section>
</template>
