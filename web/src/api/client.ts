import { useAuthStore } from '../stores/auth';
import type {
  AccessPath,
  AccessPathList,
  AccessPathManagers,
  CreateRepositoryRequest,
  ErrorEnvelope,
  Group,
  GroupDetail,
  GroupList,
  GroupMembers,
  Permission,
  Repository,
  RepositoryCreated,
  RepositoryList,
  RepositoryTree,
  RoleList,
  Session,
  SetupResult,
  SetupStatus,
  SystemInfo,
  User,
  UserAccessPaths,
  UserGroups,
  UserList,
  UserRoles,
} from './types';

/** All management endpoints live under this prefix (plan D2). */
export const API_BASE = '/svnadmin/api/v1';

/** Machine readable part of an error response (`{ error: ... }`). */
export interface ApiErrorBody {
  code: string;
  message: string;
  details?: Record<string, string>;
}

/** Error carrying the HTTP status and the API error code. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  /** The `error` object of the response envelope, when the server sent one. */
  readonly body: ApiErrorBody | null;

  constructor(status: number, code: string, message: string, body: ApiErrorBody | null = null) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.body = body;
  }

  /** True for a transport failure where no HTTP response was received. */
  get isNetworkError(): boolean {
    return this.status === 0;
  }
}

type UnauthorizedHandler = () => void;

let unauthorizedHandler: UnauthorizedHandler | null = null;

/**
 * Registers the 401 hook. main.ts wires it to "clear the token and go to
 * /login?expired=1" so this module does not import the router (avoids a cycle).
 */
export function onUnauthorized(handler: UnauthorizedHandler): void {
  unauthorizedHandler = handler;
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
  /** Set to false for the unauthenticated endpoints (POST /session). */
  auth?: boolean;
  /**
   * Set to true when the caller handles 401 itself (the router guard, the
   * login form, logout). It keeps the global hook from racing the caller's
   * own navigation.
   */
  suppressUnauthorized?: boolean;
}

function errorFromEnvelope(status: number, payload: unknown): ApiError {
  const envelope = payload as Partial<ErrorEnvelope> | null;

  if (envelope && typeof envelope === 'object' && envelope.error) {
    const body: ApiErrorBody = {
      code: envelope.error.code ?? 'error',
      message: envelope.error.message ?? 'request failed',
      details: envelope.error.details,
    };

    return new ApiError(status, body.code, body.message, body);
  }

  return new ApiError(status, 'error', `request failed with status ${status}`);
}

/**
 * Thin fetch wrapper.
 *
 * - adds `Authorization: Bearer <token>` when a token is stored;
 * - clears the token and fires the 401 hook on `401`;
 * - never reads or writes cookies (`credentials: 'omit'`).
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal, auth = true, suppressUnauthorized = false } = options;

  const headers: Record<string, string> = { Accept: 'application/json' };

  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }

  if (auth) {
    const token = useAuthStore().token;
    if (token !== '') {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  let response: Response;

  try {
    response = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
      credentials: 'omit',
      cache: 'no-store',
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw error;
    }

    throw new ApiError(0, 'network_error', error instanceof Error ? error.message : 'network error');
  }

  if (response.status === 401 && auth && !suppressUnauthorized) {
    useAuthStore().clear();
    unauthorizedHandler?.();
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  const payload: unknown = text === '' ? null : safeParse(text);

  if (!response.ok) {
    throw errorFromEnvelope(response.status, payload);
  }

  return payload as T;
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

/** Builds `?a=1&b=2`, skipping empty values. */
export function buildQuery(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '') {
      continue;
    }

    search.set(key, String(value));
  }

  const query = search.toString();

  return query === '' ? '' : `?${query}`;
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

export function createSession(username: string, password: string): Promise<Session> {
  return request<Session>('/session', {
    method: 'POST',
    body: { username, password },
    auth: false,
    suppressUnauthorized: true,
  });
}

/** Returns the current session (no token). The caller handles 401/503. */
export function fetchSession(): Promise<Session> {
  return request<Session>('/session', { suppressUnauthorized: true });
}

export function destroySession(): Promise<void> {
  return request<void>('/session', { method: 'DELETE', suppressUnauthorized: true });
}

/**
 * First-run diagnostics (no token). The reason tells the UI why setup is still
 * pending; `users` is only populated while setup is pending.
 */
export function getSetupStatus(): Promise<SetupStatus> {
  return request<SetupStatus>('/setup/status', {
    auth: false,
    suppressUnauthorized: true,
  });
}

/**
 * Creates the first administrator (no token). `password` may be an empty
 * string when the chosen account already exists: the API then reuses the
 * stored credential.
 */
export function createAdmin(username: string, password: string, token: string): Promise<SetupResult> {
  return request<SetupResult>('/setup/admin', {
    method: 'POST',
    body: { username, password, token },
    auth: false,
    suppressUnauthorized: true,
  });
}

// ---------------------------------------------------------------------------
// Read-only endpoints (P1)
// ---------------------------------------------------------------------------

export function getSystemInfo(): Promise<SystemInfo> {
  return request<SystemInfo>('/system/info');
}

export function listRepositories(page: number, size: number, q = ''): Promise<RepositoryList> {
  return request<RepositoryList>(`/repos${buildQuery({ q, page, size })}`);
}

/**
 * Repository writes (P4).
 *
 * `createRepository` runs `svnadmin create`, the optional layout commit and the
 * optional `<repo>:/` access path in one request. `deleteRepository` moves the
 * repository into the trash and requires the exact name in `confirm`.
 */
export function createRepository(payload: CreateRepositoryRequest): Promise<RepositoryCreated> {
  return request<RepositoryCreated>('/repos', { method: 'POST', body: payload });
}

export function deleteRepository(name: string, confirm: string): Promise<void> {
  return request<void>(`/repos/${encodeURIComponent(name)}${buildQuery({ confirm })}`, {
    method: 'DELETE',
  });
}

export function getRepositoryInfo(name: string): Promise<Repository> {
  return request<Repository>(`/repos/${encodeURIComponent(name)}/info`);
}

/** `rev` is passed through as given: empty means HEAD, otherwise a revision. */
export function getRepositoryTree(
  name: string,
  path = '',
  rev: string | number | undefined = undefined,
  page = 1,
  size = 100,
): Promise<RepositoryTree> {
  return request<RepositoryTree>(
    `/repos/${encodeURIComponent(name)}/tree${buildQuery({ path, rev, page, size })}`,
  );
}

export function listUsers(page: number, size: number, q = ''): Promise<UserList> {
  return request<UserList>(`/users${buildQuery({ q, page, size })}`);
}

export function getUser(name: string): Promise<User> {
  return request<User>(`/users/${encodeURIComponent(name)}`);
}

// ---------------------------------------------------------------------------
// User writes (P2)
// ---------------------------------------------------------------------------

export function createUser(username: string, password: string): Promise<User> {
  return request<User>('/users', { method: 'POST', body: { username, password } });
}

export function deleteUser(name: string): Promise<void> {
  return request<void>(`/users/${encodeURIComponent(name)}`, { method: 'DELETE' });
}

export function changeUserPassword(name: string, password: string): Promise<void> {
  return request<void>(`/users/${encodeURIComponent(name)}/password`, {
    method: 'PUT',
    body: { password },
  });
}

export function getUserRoles(name: string): Promise<UserRoles> {
  return request<UserRoles>(`/users/${encodeURIComponent(name)}/roles`);
}

export function setUserRoles(name: string, roles: string[]): Promise<UserRoles> {
  return request<UserRoles>(`/users/${encodeURIComponent(name)}/roles`, {
    method: 'PUT',
    body: { roles },
  });
}

export function getUserGroups(name: string): Promise<UserGroups> {
  return request<UserGroups>(`/users/${encodeURIComponent(name)}/groups`);
}

export function setUserGroups(name: string, groups: string[]): Promise<UserGroups> {
  return request<UserGroups>(`/users/${encodeURIComponent(name)}/groups`, {
    method: 'PUT',
    body: { groups },
  });
}

/**
 * Effective access of an account: direct user entries, entries inherited
 * through its groups and the global `*` entry.
 */
export function getUserAccessPaths(name: string): Promise<UserAccessPaths> {
  return request<UserAccessPaths>(`/users/${encodeURIComponent(name)}/access-paths`);
}

export function listGroups(page: number, size: number, q = ''): Promise<GroupList> {
  return request<GroupList>(`/groups${buildQuery({ q, page, size })}`);
}

export function getGroup(name: string): Promise<GroupDetail> {
  return request<GroupDetail>(`/groups/${encodeURIComponent(name)}`);
}

export function createGroup(name: string): Promise<Group> {
  return request<Group>('/groups', { method: 'POST', body: { name } });
}

export function deleteGroup(name: string): Promise<void> {
  return request<void>(`/groups/${encodeURIComponent(name)}`, { method: 'DELETE' });
}

export function getGroupMembers(name: string): Promise<GroupMembers> {
  return request<GroupMembers>(`/groups/${encodeURIComponent(name)}/members`);
}

export function setGroupMembers(name: string, users: string[]): Promise<GroupMembers> {
  return request<GroupMembers>(`/groups/${encodeURIComponent(name)}/members`, {
    method: 'PUT',
    body: { users },
  });
}

export function listAccessPaths(page: number, size: number, q = ''): Promise<AccessPathList> {
  return request<AccessPathList>(`/access-paths${buildQuery({ q, page, size })}`);
}

export function createAccessPath(path: string): Promise<AccessPath> {
  return request<AccessPath>('/access-paths', { method: 'POST', body: { path } });
}

/** The path segment is the URL-safe base64 encoding of e.g. `repo:/trunk`. */
export function getAccessPath(encodedPath: string): Promise<AccessPath> {
  return request<AccessPath>(`/access-paths/${encodeURIComponent(encodedPath)}/acl`);
}

export function deleteAccessPath(encodedPath: string): Promise<void> {
  return request<void>(`/access-paths/${encodeURIComponent(encodedPath)}`, { method: 'DELETE' });
}

/** `principal` is a user name, `@group` or `*`; `none` removes the entry. */
export function setAccessPathPermission(
  encodedPath: string,
  principal: string,
  permission: Permission,
): Promise<AccessPath> {
  return request<AccessPath>(
    `/access-paths/${encodeURIComponent(encodedPath)}/acl/${encodeURIComponent(principal)}`,
    { method: 'PUT', body: { permission } },
  );
}

export function deleteAccessPathPermission(encodedPath: string, principal: string): Promise<void> {
  return request<void>(
    `/access-paths/${encodeURIComponent(encodedPath)}/acl/${encodeURIComponent(principal)}`,
    { method: 'DELETE' },
  );
}

export function getAccessPathManagers(encodedPath: string): Promise<AccessPathManagers> {
  return request<AccessPathManagers>(`/access-paths/${encodeURIComponent(encodedPath)}/managers`);
}

export function setAccessPathManagers(encodedPath: string, users: string[]): Promise<AccessPathManagers> {
  return request<AccessPathManagers>(`/access-paths/${encodeURIComponent(encodedPath)}/managers`, {
    method: 'PUT',
    body: { users },
  });
}

export function listRoles(): Promise<RoleList> {
  return request<RoleList>('/roles');
}
