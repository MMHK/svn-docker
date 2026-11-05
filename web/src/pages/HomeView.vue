<script setup lang="ts">
import { RouterLink } from 'vue-router';
import { t } from '../i18n';
import { NAV_ITEMS } from '../navigation';
import { useAuthStore } from '../stores/auth';

const auth = useAuthStore();
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

    <div class="space-y-3">
      <h2 class="text-sm font-semibold">{{ t('home.shortcuts') }}</h2>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <RouterLink
          v-for="item in NAV_ITEMS"
          :key="item.name"
          :to="{ name: item.name }"
          class="rounded border border-slate-200 bg-white p-4 transition hover:border-sky-400 hover:shadow-sm dark:border-slate-800 dark:bg-slate-900 dark:hover:border-sky-600"
        >
          <p class="font-medium text-slate-900 dark:text-white">{{ t(item.labelKey) }}</p>
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ t(item.descriptionKey) }}</p>
        </RouterLink>
      </div>
    </div>
  </section>
</template>
