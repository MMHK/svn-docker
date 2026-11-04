import { ref } from 'vue';
import enUS from './en_US.json';
import zhCN from './zh_CN.json';

/**
 * Minimal locale helper for P0. vue-i18n is intentionally not pulled in yet:
 * the plan upgrades i18n in P1 and only zh_CN / en_US are supported (D15/D21).
 */
export type Locale = 'zh_CN' | 'en_US';

export const SUPPORTED_LOCALES: readonly Locale[] = ['zh_CN', 'en_US'];

export const LOCALE_STORAGE_KEY = 'svnadmin.locale';

const dictionaries: Record<Locale, unknown> = { zh_CN: zhCN, en_US: enUS };

export const DEFAULT_LOCALE: Locale = 'zh_CN';

function isLocale(value: unknown): value is Locale {
  return value === 'zh_CN' || value === 'en_US';
}

function readStoredLocale(): Locale {
  try {
    const stored = localStorage.getItem(LOCALE_STORAGE_KEY);
    if (isLocale(stored)) {
      return stored;
    }
  } catch {
    // localStorage can be unavailable (private mode / disabled storage).
  }

  return DEFAULT_LOCALE;
}

export const locale = ref<Locale>(readStoredLocale());

export function setLocale(next: Locale): void {
  locale.value = next;
  document.documentElement.lang = next;

  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, next);
  } catch {
    // Ignore: the in-memory locale still applies for this session.
  }
}

function lookup(dictionary: unknown, key: string): string | undefined {
  let node: unknown = dictionary;

  for (const part of key.split('.')) {
    if (typeof node !== 'object' || node === null) {
      return undefined;
    }

    node = (node as Record<string, unknown>)[part];
  }

  return typeof node === 'string' ? node : undefined;
}

/** Translates a dotted key, falling back to zh_CN and then to the key itself. */
export function t(key: string): string {
  return (
    lookup(dictionaries[locale.value], key) ??
    lookup(dictionaries[DEFAULT_LOCALE], key) ??
    key
  );
}

/** Applies the stored locale to <html lang> once at start-up. */
export function initLocale(): void {
  document.documentElement.lang = locale.value;
}
