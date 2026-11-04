/** DTOs mirrored from api/pkg/view. Keep the JSON names identical. */

export interface PathInfo {
  path: string;
  exists: boolean;
  isDir: boolean;
  readable: boolean;
  writable: boolean;
}

export interface SystemInfo {
  version: string;
  goVersion: string;
  platform: string;
  svnParentPath: PathInfo;
  svnAuthFile: PathInfo;
  svnAccessFile: PathInfo;
  docsEnabled: boolean;
  defaultLocale: string;
  serverTime: string;
}

export interface ErrorDetail {
  code: string;
  message: string;
  details?: Record<string, string>;
}

export interface ErrorEnvelope {
  error: ErrorDetail;
}

/** Shared pagination block embedded in every list response. */
export interface ListMeta {
  page: number;
  size: number;
  total: number;
}

export interface SessionUser {
  name: string;
  roles: string[];
}

/** GET /session omits the token; POST /session and /session/refresh return it. */
export interface Session {
  token?: string;
  expiresAt: string;
  idleExpiresAt: string;
  me: SessionUser;
  roles: string[];
  permissions: string[];
}

export interface SetupResult {
  username: string;
  role: string;
  accountCreated: boolean;
  initializedFiles: string[];
}

/** State of one bootstrap-related file (GET /setup/status). */
export interface SetupFileStatus {
  exists: boolean;
  readable: boolean;
  writable: boolean;
}

/** Credential file state, which also carries the account count. */
export interface SetupAuthFileStatus extends SetupFileStatus {
  userCount: number;
}

/** Role file state; `administratorPresent` says whether setup is already done. */
export interface SetupRoleFileStatus extends SetupFileStatus {
  administratorPresent: boolean;
}

/**
 * GET /setup/status payload. `reason` is one of `missing_auth_file`,
 * `missing_access_file`, `no_admin` or `''` (setup completed). `users` is only
 * populated while setup is pending.
 */
export interface SetupStatus {
  setupRequired: boolean;
  reason: string;
  authFile: SetupAuthFileStatus;
  accessFile: SetupFileStatus;
  roleFile: SetupRoleFileStatus;
  users: string[];
  canCreateAccount: boolean;
}

export interface User {
  name: string;
  comment?: string;
  roles: string[];
  groups: string[];
  createdAt?: string;
  updatedAt?: string;
}

export interface UserList extends ListMeta {
  items: User[];
}

/** GET/PUT /users/{name}/roles payload. */
export interface UserRoles {
  name: string;
  roles: string[];
}

/** GET/PUT /users/{name}/groups payload. */
export interface UserGroups {
  name: string;
  groups: string[];
}

export interface Group {
  name: string;
  description?: string;
  members: string[];
}

export interface GroupDetail extends Group {
  accessPaths: string[];
}

export interface GroupList extends ListMeta {
  items: Group[];
}

/** GET/PUT /groups/{name}/members payload. */
export interface GroupMembers {
  name: string;
  users: string[];
}

export interface Repository {
  name: string;
  parent: string;
  url?: string;
  isRepository: boolean;
  headRevision: number;
  lastChangedRev: number;
  lastChangedAuthor?: string;
  lastChangedAt?: string;
}

export interface RepositoryList extends ListMeta {
  items: Repository[];
}

export type RepositoryStructure = 'none' | 'single' | 'multi';

/** POST /repos body. `projectName` is only allowed with structure "multi". */
export interface CreateRepositoryRequest {
  name: string;
  structure?: RepositoryStructure;
  projectName?: string;
  accessPath?: boolean;
}

/** POST /repos 201 payload. */
export interface RepositoryCreated {
  name: string;
  parent: string;
  url: string;
  createdStructure: RepositoryStructure;
}

export interface RepositoryEntry {
  name: string;
  kind: 'dir' | 'file';
  size?: number;
  revision: number;
  author?: string;
  date?: string;
}

/** GET /repos/{repo}/tree payload. */
export interface RepositoryTree extends ListMeta {
  path: string;
  revision: number;
  entries: RepositoryEntry[];
}

export type Permission = 'none' | 'r' | 'rw';

export interface ACLEntry {
  principal: string;
  permission: Permission;
}

export interface AccessPath {
  path: string;
  acl: ACLEntry[];
  managers: string[];
}

export interface AccessPathList extends ListMeta {
  items: AccessPath[];
}

/** GET/PUT /access-paths/{path}/managers payload. */
export interface AccessPathManagers {
  path: string;
  users: string[];
}

/** One effective access source of an account (GET /users/{name}/access-paths). */
export interface UserAccessPath {
  path: string;
  permission: Permission;
  /** True for a `@group` or `*` entry, false for a direct user entry. */
  inherited: boolean;
  source: 'user' | 'group' | 'global';
  group?: string;
}

/** GET /users/{name}/access-paths payload. */
export interface UserAccessPaths {
  name: string;
  items: UserAccessPath[];
}

export interface Role {
  name: string;
  description: string;
}

export interface RoleList {
  items: Role[];
}
