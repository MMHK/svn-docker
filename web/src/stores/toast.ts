import { defineStore } from 'pinia';

export type ToastTone = 'success' | 'error';

export interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

let nextId = 0;

/**
 * Tiny toast queue. Messages are already translated by the caller so the store
 * stays free of i18n lookups and the toast survives a route change (for example
 * the forced sign-out after changing one's own password).
 */
export const useToastStore = defineStore('toast', {
  state: () => ({
    items: [] as ToastItem[],
  }),
  actions: {
    push(message: string, tone: ToastTone = 'success', ttl = 4000): void {
      const id = (nextId += 1);
      this.items.push({ id, message, tone });

      if (ttl > 0) {
        window.setTimeout(() => this.dismiss(id), ttl);
      }
    },

    dismiss(id: number): void {
      this.items = this.items.filter((item) => item.id !== id);
    },
  },
});
