import { defineStore } from 'pinia';
import type { Session } from '../api/types';

/**
 * Auth state.
 *
 * The bearer token lives in localStorage under `svnadmin.token` and is sent as
 * `Authorization: Bearer <token>` by the API client. Cookies are deliberately
 * not used (plan D9), so there is no CSRF token either.
 */
export const TOKEN_STORAGE_KEY = 'svnadmin.token';

function readToken(): string {
  try {
    return localStorage.getItem(TOKEN_STORAGE_KEY) ?? '';
  } catch {
    return '';
  }
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: readToken(),
    expiresAt: '',
    idleExpiresAt: '',
    me: null as { name: string; roles: string[] } | null,
    roles: [] as string[],
    permissions: [] as string[],
    /** True once GET /session confirmed the token for this tab. */
    validated: false,
  }),
  getters: {
    isAuthenticated: (state): boolean => state.token !== '',
    userName: (state): string => state.me?.name ?? '',
    isAdministrator: (state): boolean => state.roles.includes('Administrator'),
  },
  actions: {
    setToken(token: string): void {
      this.token = token;

      try {
        localStorage.setItem(TOKEN_STORAGE_KEY, token);
      } catch {
        // The in-memory token still works for the current tab.
      }
    },

    /** Applies POST /session, GET /session or POST /session/refresh output. */
    applySession(session: Session): void {
      if (typeof session.token === 'string' && session.token !== '') {
        this.setToken(session.token);
      }

      this.me = session.me ?? null;
      this.roles = session.roles ?? session.me?.roles ?? [];
      this.permissions = session.permissions ?? [];
      this.expiresAt = session.expiresAt ?? '';
      this.idleExpiresAt = session.idleExpiresAt ?? '';
      this.validated = true;
    },

    clear(): void {
      this.token = '';
      this.expiresAt = '';
      this.idleExpiresAt = '';
      this.me = null;
      this.roles = [];
      this.permissions = [];
      this.validated = false;

      try {
        localStorage.removeItem(TOKEN_STORAGE_KEY);
      } catch {
        // Nothing else to do: the in-memory token is already gone.
      }
    },

    /** True when the session carries `module:action` (Administrator wins). */
    can(permission: string): boolean {
      if (this.roles.includes('Administrator')) {
        return true;
      }

      return this.permissions.includes(permission);
    },
  },
});
