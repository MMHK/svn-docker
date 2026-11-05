/**
 * Sidebar/overview entries. All authenticated users can access all sections.
 */
export interface NavItem {
  name: string;
  labelKey: string;
  descriptionKey: string;
}

export const NAV_ITEMS: NavItem[] = [
  {
    name: 'repos',
    labelKey: 'nav.repos',
    descriptionKey: 'home.reposHint',
  },
  {
    name: 'users',
    labelKey: 'nav.users',
    descriptionKey: 'home.usersHint',
  },
  {
    name: 'groups',
    labelKey: 'nav.groups',
    descriptionKey: 'home.groupsHint',
  },
  {
    name: 'access-paths',
    labelKey: 'nav.accessPaths',
    descriptionKey: 'home.accessPathsHint',
  },
  {
    name: 'system',
    labelKey: 'nav.system',
    descriptionKey: 'home.systemHint',
  },
];
