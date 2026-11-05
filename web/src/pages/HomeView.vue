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
      <p class="mt-1 text-sm text-[#656d76] dark:text-[#c9d1d9]">
        {{ t('home.welcome') }}
        <span v-if="auth.userName" class="font-medium text-[#1f2328] dark:text-[#e6edf3]">{{ auth.userName }}</span>
      </p>
    </div>

    <div class="space-y-3">
      <h2 class="text-sm font-semibold">{{ t('home.shortcuts') }}</h2>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <RouterLink
          v-for="item in NAV_ITEMS"
          :key="item.name"
          :to="{ name: item.name }"
          class="rounded-md border border-[#d0d7de] bg-[#ffffff] p-4 transition hover:border-[#0969da] hover:shadow-sm dark:border-[#30363d] dark:bg-[#161b22] dark:hover:border-[#2f81f7]"
        >
          <p class="font-medium text-[#1f2328] dark:text-[#e6edf3]">{{ t(item.labelKey) }}</p>
          <p class="mt-1 text-xs text-[#656d76] dark:text-[#8b949e]">{{ t(item.descriptionKey) }}</p>
        </RouterLink>
      </div>
    </div>
  </section>
</template>
