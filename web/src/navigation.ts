/**
 * Sidebar/overview entries. `permission` is the `module:action` string the
 * session carries (plan section 4.3); entries the session cannot use are
 * hidden. The API still enforces the matrix, so a deep link renders 403.
 */
export interface NavItem {
  name: string;
  labelKey: string;
  descriptionKey: string;
  permission: string;
}

export const NAV_ITEMS: NavItem[] = [
  {
    name: 'repos',
    labelKey: 'nav.repos',
    descriptionKey: 'home.reposHint',
    permission: 'repository:view',
  },
  {
    name: 'users',
    labelKey: 'nav.users',
    descriptionKey: 'home.usersHint',
    permission: 'user:view',
  },
  {
    name: 'groups',
    labelKey: 'nav.groups',
    descriptionKey: 'home.groupsHint',
    permission: 'group:view',
  },
  {
    name: 'access-paths',
    labelKey: 'nav.accessPaths',
    descriptionKey: 'home.accessPathsHint',
    permission: 'access-path:view',
  },
  {
    name: 'roles',
    labelKey: 'nav.roles',
    descriptionKey: 'home.rolesHint',
    permission: 'role:view',
  },
  {
    name: 'system',
    labelKey: 'nav.system',
    descriptionKey: 'home.systemHint',
    permission: 'settings:view',
  },
];
