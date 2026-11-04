<script setup lang="ts">
import { computed, ref } from 'vue';
import { RouterLink, RouterView, useRouter } from 'vue-router';
import { destroySession } from '../api/client';
import { locale, setLocale, t, type Locale } from '../i18n';
import { NAV_ITEMS, type NavItem } from '../navigation';
import { useAuthStore } from '../stores/auth';
import { theme, toggleTheme } from '../theme';

const auth = useAuthStore();
const router = useRouter();

const signingOut = ref(false);

/**
 * Navigation is filtered by the `module:action` permissions returned by
 * GET /session (plan section 4.3). Hiding an entry is cosmetic: the API still
 * answers 403 and the page renders that state.
 */
const visibleItems = computed<NavItem[]>(() => NAV_ITEMS.filter((item) => auth.can(item.permission)));

const themeLabel = computed(() => (theme.value === 'dark' ? t('theme.light') : t('theme.dark')));

function changeLocale(event: Event): void {
  const value = (event.target as HTMLSelectElement).value;
  if (value === 'zh_CN' || value === 'en_US') {
    setLocale(value as Locale);
  }
}

async function signOut(): Promise<void> {
  if (signingOut.value) {
    return;
  }

  signingOut.value = true;

  try {
    await destroySession();
  } catch {
    // The token may already be invalid; clearing the client state is enough.
  } finally {
    auth.clear();
    signingOut.value = false;
    await router.push({ name: 'login' });
  }
}
</script>

<template>
  <div class="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
    <header class="border-b border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
      <div class="flex w-full flex-wrap items-center gap-3 px-4 py-3">
        <RouterLink to="/" class="flex items-baseline gap-2">
          <span class="text-lg font-semibold tracking-tight">{{ t('app.title') }}</span>
          <span class="hidden text-xs text-slate-500 sm:inline dark:text-slate-400">{{ t('app.subtitle') }}</span>
        </RouterLink>

        <div class="ml-auto flex flex-wrap items-center gap-2">
          <span v-if="auth.userName" class="text-sm text-slate-600 dark:text-slate-300">
            {{ t('nav.signedInAs') }} <span class="font-medium text-slate-900 dark:text-white">{{ auth.userName }}</span>
          </span>

          <select
            class="rounded border border-slate-300 bg-white px-2 py-1 text-sm text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200"
            :value="locale"
            :aria-label="t('nav.language')"
            @change="changeLocale"
          >
            <option value="zh_CN">繁體中文</option>
            <option value="en_US">English</option>
          </select>

          <button
            type="button"
            class="rounded border border-slate-300 px-2 py-1 text-sm hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800"
            :title="themeLabel"
            @click="toggleTheme"
          >
            {{ themeLabel }}
          </button>

          <button
            type="button"
            class="rounded border border-slate-300 px-2 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
            :disabled="signingOut"
            @click="signOut"
          >
            {{ t('nav.logout') }}
          </button>
        </div>
      </div>
    </header>

    <div class="flex flex-col md:flex-row">
      <aside class="shrink-0 border-b border-slate-200 bg-white px-2 py-3 md:w-56 md:border-b-0 md:border-r dark:border-slate-800 dark:bg-slate-900">
        <nav class="flex flex-row flex-wrap gap-1 md:flex-col" :aria-label="t('nav.menu')">
          <RouterLink
            :to="{ name: 'home' }"
            class="rounded px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
            active-class="bg-slate-100 font-medium text-slate-900 dark:bg-slate-800 dark:text-white"
            exact-active-class="bg-slate-100 font-medium text-slate-900 dark:bg-slate-800 dark:text-white"
          >
            {{ t('nav.home') }}
          </RouterLink>
          <RouterLink
            v-for="item in visibleItems"
            :key="item.name"
            :to="{ name: item.name }"
            class="rounded px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
            active-class="bg-slate-100 font-medium text-slate-900 dark:bg-slate-800 dark:text-white"
          >
            {{ t(item.labelKey) }}
          </RouterLink>
        </nav>
      </aside>

      <main class="min-w-0 flex-1 px-4 py-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
