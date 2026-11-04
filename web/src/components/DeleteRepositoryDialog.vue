<script setup lang="ts">
import { ref } from 'vue';
import { ApiError, deleteRepository } from '../api/client';
import { describeError, toError } from '../api/errors';
import ModalDialog from './ModalDialog.vue';
import { t } from '../i18n';
import { useToastStore } from '../stores/toast';

/**
 * Confirmation gate for DELETE /repos/{repo} (plan D7/D20).
 *
 * The repository is only moved into `data/trash/<repo>-<UTC ts>` and kept for
 * 30 days, so the copy says so explicitly instead of implying a hard delete.
 * The caller passes the exact repository name; the submit button stays disabled
 * until the operator repeats it.
 */
const props = defineProps<{
  open: boolean;
  name: string;
}>();

const emit = defineEmits<{
  (event: 'close'): void;
  (event: 'deleted'): void;
}>();

const toast = useToastStore();

const confirm = ref('');
const error = ref('');
const saving = ref(false);

function reset(): void {
  confirm.value = '';
  error.value = '';
  saving.value = false;
}

function close(): void {
  if (saving.value) {
    return;
  }

  reset();
  emit('close');
}

/** 403 (feature disabled / no role), 409 (cross-filesystem) and 422 get copy. */
function deleteErrorMessage(cause: unknown): string {
  if (!(cause instanceof ApiError)) {
    return t('repos.delete.failed');
  }

  switch (cause.status) {
    case 0:
      return t('common.networkErrorHint');
    case 403:
      return t('repos.delete.forbidden');
    case 404:
      return t('common.notFoundHint');
    case 409:
      return t('repos.delete.crossDevice');
    case 422:
      return cause.message;
    default:
      return `${t('repos.delete.failed')}: ${describeError(toError(cause)).detail}`;
  }
}

async function submit(): Promise<void> {
  if (saving.value || confirm.value.trim() !== props.name) {
    return;
  }

  saving.value = true;
  error.value = '';

  try {
    await deleteRepository(props.name, confirm.value.trim());
    toast.push(t('repos.delete.success'));
    reset();
    emit('deleted');
  } catch (cause) {
    error.value = deleteErrorMessage(cause);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <ModalDialog :open="open" :title="t('repos.delete.title')" @close="close">
    <form id="delete-repository-form" class="space-y-4" @submit.prevent="submit">
      <p class="text-sm text-red-700 dark:text-red-300">{{ t('repos.delete.hint') }}</p>

      <p class="rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-200">
        {{ t('repos.delete.trashNote') }}
      </p>

      <p class="text-sm">
        {{ t('repos.delete.confirmLabel') }}
        <span class="font-mono font-semibold">{{ name }}</span>
      </p>

      <div>
        <label for="delete-repository-confirm" class="block text-sm font-medium">
          {{ t('repos.delete.confirmField') }}
        </label>
        <input
          id="delete-repository-confirm"
          v-model="confirm"
          type="text"
          autocomplete="off"
          spellcheck="false"
          class="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm dark:border-slate-700 dark:bg-slate-800"
        />
      </div>

      <p
        v-if="error"
        class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200"
        role="alert"
      >
        {{ error }}
      </p>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button
          type="button"
          class="rounded border border-slate-300 px-3 py-1 text-sm hover:bg-slate-100 disabled:opacity-50 dark:border-slate-700 dark:hover:bg-slate-800"
          :disabled="saving"
          @click="close"
        >
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="delete-repository-form"
          class="rounded bg-red-600 px-3 py-1 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
          :disabled="saving || confirm.trim() !== name"
        >
          {{ saving ? t('repos.delete.deleting') : t('repos.delete.submit') }}
        </button>
      </div>
    </template>
  </ModalDialog>
</template>
