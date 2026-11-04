<script setup lang="ts">
import { ref } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import { ApiError, createAdmin } from '../api/client';
import { t } from '../i18n';

const router = useRouter();

const username = ref('');
const bootstrapToken = ref('');
const submitting = ref(false);
const error = ref('');
const created = ref('');

function messageFor(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('setup.networkError');
  }

  switch (cause.status) {
    case 0:
      return t('setup.networkError');
    case 403:
      return t('setup.invalidToken');
    case 404:
      return t('setup.unknownAccount');
    case 409:
      return t('setup.alreadySetUp');
    case 422:
      return t('setup.missingFields');
    default:
      return `${t('setup.failed')} (${cause.status})`;
  }
}

async function submit(): Promise<void> {
  error.value = '';
  created.value = '';
  submitting.value = true;

  try {
    const result = await createAdmin(username.value, bootstrapToken.value);
    created.value = result.username;
    bootstrapToken.value = '';
  } catch (cause) {
    error.value = messageFor(cause);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <section class="mx-auto mt-10 max-w-md px-4">
    <h1 class="text-xl font-semibold">{{ t('setup.title') }}</h1>
    <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">{{ t('setup.intro') }}</p>

    <div
      v-if="created !== ''"
      class="mt-6 rounded border border-emerald-300 bg-emerald-50 px-4 py-4 text-sm text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-200"
      role="status"
    >
      <p class="font-medium">{{ t('setup.success') }}</p>
      <p class="mt-1">{{ t('setup.successHint') }}</p>
      <button
        type="button"
        class="mt-3 rounded bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700"
        @click="router.push({ name: 'login' })"
      >
        {{ t('setup.gotoLogin') }}
      </button>
    </div>

    <form
      v-else
      class="mt-6 space-y-4 rounded border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900"
      @submit.prevent="submit"
    >
      <div>
        <label for="setup-username" class="block text-sm font-medium">{{ t('setup.username') }}</label>
        <p class="text-xs text-slate-500 dark:text-slate-400">{{ t('setup.usernameHint') }}</p>
        <input
          id="setup-username"
          v-model="username"
          name="username"
          type="text"
          autocomplete="username"
          required
          class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
        />
      </div>

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

      <p v-if="error" class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200" role="alert">
        {{ error }}
      </p>

      <button
        type="submit"
        class="w-full rounded bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50"
        :disabled="submitting"
      >
        {{ submitting ? t('setup.submitting') : t('setup.submit') }}
      </button>

      <p class="text-center text-xs text-slate-500 dark:text-slate-400">
        <RouterLink to="/login" class="text-sky-600 hover:underline dark:text-sky-400">
          {{ t('setup.backToLogin') }}
        </RouterLink>
      </p>
    </form>
  </section>
</template>
