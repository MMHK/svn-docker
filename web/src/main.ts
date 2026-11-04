import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import { onUnauthorized } from './api/client';
import { initLocale } from './i18n';
import { router } from './router';
import { useAuthStore } from './stores/auth';
import { initTheme } from './theme';
import './styles.css';

const pinia = createPinia();
const app = createApp(App);

app.use(pinia);

initTheme();
initLocale();

// A 401 means the session token is gone or revoked: drop it and send the user
// to the login page with an explicit marker (plan section 4.4). Registered here
// so api/client.ts does not import the router (avoids a module cycle).
onUnauthorized(() => {
  const auth = useAuthStore(pinia);
  auth.clear();

  const current = router.currentRoute.value;

  if (current.meta.public !== true) {
    void router.replace({ name: 'login', query: { expired: '1' } });
  }
});

app.use(router);
app.mount('#app');
