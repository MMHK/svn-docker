import { ref } from 'vue';

/**
 * Theme is stored in localStorage and applied as <html data-theme="light|dark">
 * (plan D9: no cookies anywhere in the management UI).
 */
export type Theme = 'light' | 'dark';

export const THEME_STORAGE_KEY = 'svnadmin.theme';

export const theme = ref<Theme>('light');

function isTheme(value: unknown): value is Theme {
  return value === 'light' || value === 'dark';
}

function readStoredTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    if (isTheme(stored)) {
      return stored;
    }
  } catch {
    // Ignore unavailable storage and fall back to the OS preference.
  }

  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  return 'light';
}

export function applyTheme(next: Theme): void {
  theme.value = next;

  const root = document.documentElement;
  root.dataset.theme = next;
  root.style.colorScheme = next;

  try {
    localStorage.setItem(THEME_STORAGE_KEY, next);
  } catch {
    // Ignore: the theme still applies for this page load.
  }
}

export function initTheme(): void {
  applyTheme(readStoredTheme());
}

export function toggleTheme(): void {
  applyTheme(theme.value === 'dark' ? 'light' : 'dark');
}
