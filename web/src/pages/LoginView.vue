<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ApiError, createSession, fetchSession } from '../api/client';
import { t } from '../i18n';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();
const route = useRoute();
const router = useRouter();

const username = ref('');
const password = ref('');
const submitting = ref(false);
const error = ref('');

const expired = computed(() => route.query.expired === '1');

/** Same-origin absolute paths only (avoids an open redirect). */
const redirect = computed<string | null>(() => {
  const value = route.query.redirect;
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) {
    return null;
  }

  if (value.startsWith('/login') || value.startsWith('/setup')) {
    return null;
  }

  return value;
});

function isSetupRequired(cause: unknown): boolean {
  return cause instanceof ApiError && (cause.code === 'setup_required' || cause.status === 503);
}

function messageFor(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('login.networkError');
  }

  switch (cause.status) {
    case 0:
      return t('login.networkError');
    case 401:
      return t('login.invalidCredentials');
    case 403:
      return t('login.forbidden');
    case 422:
      return t('login.missingFields');
    case 429:
      return t('login.rateLimited');
    default:
      return `${t('login.failed')} (${cause.status})`;
  }
}

async function submit(): Promise<void> {
  error.value = '';
  submitting.value = true;

  try {
    const session = await createSession(username.value, password.value);
    auth.applySession(session);
    await router.push(redirect.value === null ? { name: 'home' } : redirect.value);
  } catch (cause) {
    if (isSetupRequired(cause)) {
      await router.replace({ name: 'setup' });
      return;
    }

    error.value = messageFor(cause);
  } finally {
    submitting.value = false;
  }
}

// A fresh installation reports 503 setup_required on every guarded route, so
// probing once here takes the operator straight to the bootstrap form.
onMounted(async () => {
  if (auth.token !== '') {
    return;
  }

  try {
    await fetchSession();
  } catch (cause) {
    if (isSetupRequired(cause)) {
      await router.replace({ name: 'setup' });
    }
  }
});
</script>

<template>
  <section class="mx-auto mt-10 max-w-md px-4">
    <h1 class="text-xl font-semibold">{{ t('login.title') }}</h1>
    <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">{{ t('app.subtitle') }}</p>

    <p
      v-if="expired"
      class="mt-4 rounded border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200"
      role="status"
    >
      {{ t('login.expired') }}
    </p>

    <form
      class="mt-6 space-y-4 rounded border border-slate-200 bg-white p-6 dark:border-slate-800 dark:bg-slate-900"
      @submit.prevent="submit"
    >
      <div>
        <label for="username" class="block text-sm font-medium">{{ t('login.username') }}</label>
        <input
          id="username"
          v-model="username"
          name="username"
          type="text"
          autocomplete="username"
          required
          class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
        />
      </div>

      <div>
        <label for="password" class="block text-sm font-medium">{{ t('login.password') }}</label>
        <input
          id="password"
          v-model="password"
          name="password"
          type="password"
          autocomplete="current-password"
          required
          class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-700 dark:bg-slate-800"
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
        {{ submitting ? t('login.submitting') : t('login.submit') }}
      </button>

      <p class="text-center text-xs text-slate-500 dark:text-slate-400">
        <RouterLink to="/setup" class="text-sky-600 hover:underline dark:text-sky-400">
          {{ t('login.setupLink') }}
        </RouterLink>
      </p>
    </form>
  </section>
</template>
