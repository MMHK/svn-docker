<script setup lang="ts">
import { RouterLink } from 'vue-router';
import { t } from '../i18n';
</script>

<template>
  <section class="mx-auto mt-10 max-w-md space-y-4 px-4 text-center">
    <h1 class="text-xl font-semibold">{{ t('common.notFound') }}</h1>
    <RouterLink to="/" class="text-sm text-[#0969da] hover:underline dark:text-[#2f81f7]">
      {{ t('common.backHome') }}
    </RouterLink>
  </section>
</template>
