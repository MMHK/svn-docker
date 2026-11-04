<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink } from 'vue-router';
import { t } from '../i18n';
import { NAV_ITEMS, type NavItem } from '../navigation';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();

const shortcuts = computed<NavItem[]>(() => NAV_ITEMS.filter((item) => auth.can(item.permission)));
</script>

<template>
  <section class="space-y-6">
    <div>
      <h1 class="text-xl font-semibold">{{ t('home.title') }}</h1>
      <p class="mt-1 text-sm text-slate-600 dark:text-slate-300">
        {{ t('home.welcome') }}
        <span v-if="auth.userName" class="font-medium text-slate-900 dark:text-white">{{ auth.userName }}</span>
      </p>
    </div>

    <div class="grid gap-4 sm:grid-cols-2">
      <div class="rounded border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">{{ t('home.roles') }}</h2>
        <p v-if="auth.roles.length === 0" class="mt-2 text-sm text-slate-500 dark:text-slate-400">
          {{ t('home.noRoles') }}
        </p>
        <ul v-else class="mt-2 flex flex-wrap gap-2">
          <li
            v-for="role in auth.roles"
            :key="role"
            class="rounded bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-200"
          >
            {{ role }}
          </li>
        </ul>
      </div>

      <div class="rounded border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <h2 class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">{{ t('home.permissions') }}</h2>
        <p v-if="auth.permissions.length === 0" class="mt-2 text-sm text-slate-500 dark:text-slate-400">
          {{ t('home.noPermissions') }}
        </p>
        <ul v-else class="mt-2 flex flex-wrap gap-2">
          <li
            v-for="permission in auth.permissions"
            :key="permission"
            class="rounded bg-slate-100 px-2 py-1 font-mono text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-200"
          >
            {{ permission }}
          </li>
        </ul>
      </div>
    </div>

    <div v-if="shortcuts.length > 0" class="space-y-3">
      <h2 class="text-sm font-semibold">{{ t('home.shortcuts') }}</h2>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <RouterLink
          v-for="item in shortcuts"
          :key="item.name"
          :to="{ name: item.name }"
          class="rounded border border-slate-200 bg-white p-4 transition hover:border-sky-400 hover:shadow-sm dark:border-slate-800 dark:bg-slate-900 dark:hover:border-sky-600"
        >
          <p class="font-medium text-slate-900 dark:text-white">{{ t(item.labelKey) }}</p>
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t(item.descriptionKey) }}</p>
        </RouterLink>
      </div>
    </div>

    <p v-else class="rounded border border-dashed border-slate-300 bg-white px-4 py-6 text-center text-sm text-slate-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-400">
      {{ t('home.noModules') }}
    </p>
  </section>
</template>
