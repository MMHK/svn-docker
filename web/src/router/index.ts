import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { ApiError, fetchSession } from '../api/client';
import AppLayout from '../components/AppLayout.vue';
import AccessPathDetailView from '../pages/AccessPathDetailView.vue';
import AccessPathsView from '../pages/AccessPathsView.vue';
import GroupDetailView from '../pages/GroupDetailView.vue';
import GroupsView from '../pages/GroupsView.vue';
import HomeView from '../pages/HomeView.vue';
import LoginView from '../pages/LoginView.vue';
import NotFoundView from '../pages/NotFoundView.vue';
import ReposView from '../pages/ReposView.vue';
import RepositoryBrowseView from '../pages/RepositoryBrowseView.vue';
import SetupView from '../pages/SetupView.vue';
import SystemView from '../pages/SystemView.vue';
import UserDetailView from '../pages/UserDetailView.vue';
import UsersView from '../pages/UsersView.vue';
import { useAuthStore } from '../stores/auth';

/**
 * History mode under /svnadmin/ (plan D10). The Go SPA handler returns
 * index.html for unknown non-API paths, so deep links work without a server
 * rewrite rule.
 *
 * Everything except /login and /setup sits inside AppLayout, which renders the
 * header and the permission-aware navigation.
 */
export const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: { public: true },
  },
  {
    path: '/setup',
    name: 'setup',
    component: SetupView,
    meta: { public: true },
  },
  {
    path: '/',
    component: AppLayout,
    children: [
      { path: '', name: 'home', component: HomeView },
      { path: 'repos', name: 'repos', component: ReposView },
      {
        path: 'repositories/:repo/browse',
        name: 'repo-browse',
        component: RepositoryBrowseView,
        props: true,
      },
      { path: 'users', name: 'users', component: UsersView },
      { path: 'users/:name', name: 'user-detail', component: UserDetailView, props: true },
      { path: 'groups', name: 'groups', component: GroupsView },
      { path: 'groups/:name', name: 'group-detail', component: GroupDetailView, props: true },
      { path: 'access-paths', name: 'access-paths', component: AccessPathsView },
      { path: 'access-paths/:path', name: 'access-path-detail', component: AccessPathDetailView, props: true },
      { path: 'system', name: 'system', component: SystemView },
      { path: ':pathMatch(.*)*', name: 'not-found', component: NotFoundView },
    ],
  },
];

export const router = createRouter({
  history: createWebHistory('/svnadmin/'),
  routes,
});

/** Only same-origin absolute paths may be used as a post-login redirect. */
function safeRedirect(value: unknown): string | null {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) {
    return null;
  }

  if (value.startsWith('/login') || value.startsWith('/setup')) {
    return null;
  }

  return value;
}

/**
 * Auth guard (plan section 4.4):
 * - no token + protected route -> /login (keeping the intended path);
 * - token present but unconfirmed -> GET /session; 401 clears it, 503 sends the
 *   operator to the first-run setup form.
 */
router.beforeEach(async (to) => {
  const auth = useAuthStore();

  if (to.meta.public === true) {
    if (to.name === 'login' && auth.token !== '') {
      try {
        auth.applySession(await fetchSession());
        return { name: 'home' };
      } catch (cause) {
        if (cause instanceof ApiError && cause.status === 401) {
          auth.clear();
        }
      }
    }

    return true;
  }

  if (auth.token === '') {
    const redirect = safeRedirect(to.fullPath);

    return {
      name: 'login',
      query: redirect === null ? {} : { redirect },
    };
  }

  if (!auth.validated) {
    try {
      auth.applySession(await fetchSession());
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 401) {
        auth.clear();

        return { name: 'login', query: { expired: '1' } };
      }

      if (cause instanceof ApiError && (cause.code === 'setup_required' || cause.status === 503)) {
        auth.clear();

        return { name: 'setup' };
      }

      // Network or server failure: let the page render its own error state
      // instead of showing a blank screen.
    }
  }

  return true;
});
