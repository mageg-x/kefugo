import { createRouter, createWebHistory } from "vue-router";
import Login from "@/views/Login.vue";
import Home from "@/views/Home.vue";
import Dashboard from "@/views/Dashboard.vue";
import Visitors from "@/views/Visitors.vue";
import Staff from "@/views/Staff.vue";
import Settings from "@/views/Settings.vue";
import Inbox from "@/views/Inbox.vue";
import Snippets from "@/views/Snippets.vue";
import Profile from "@/views/Profile.vue";
import More from "@/views/More.vue";
import Apps from "@/views/Apps.vue";
import Audit from "@/views/Audit.vue";
import Knowledge from "@/views/Knowledge.vue";
import { useStore } from "@/script/store";

const routes = [
  {
    path: "/",
    redirect: "/home",
  },
  {
    path: "/login",
    name: "Login",
    component: Login,
    meta: {
      requiresAuth: false,
    },
  },
  {
    path: "/home",
    component: Home,
    meta: {
      requiresAuth: true,
    },
    children: [
      {
        path: "dashboard",
        name: "Dashboard",
        component: Dashboard,
        meta: {
          requiresAdmin: true,
        },
      },
      {
        path: "visitors",
        name: "Visitors",
        component: Visitors,
        meta: {
          requiresAdmin: true,
        },
      },
      {
        path: "staff",
        name: "Staff",
        component: Staff,
        meta: {
          requiresAdmin: true,
        },
      },
      {
        path: "apps",
        name: "Apps",
        component: Apps,
        meta: {
          requiresAdmin: true,
        },
      },
      {
        path: "settings",
        name: "Settings",
        component: Settings,
        meta: {
          requiresAdmin: true,
        },
      },
      {
        path: "audit",
        name: "Audit",
        component: Audit,
        meta: {
          requiresAdmin: true,
        },
      },
      {
        path: "inbox",
        name: "Inbox",
        component: Inbox,
        meta: {
          requiresAgent: true,
        },
      },
      {
        path: "snippets",
        name: "Snippets",
        component: Snippets,
        meta: {
          requiresAgent: true,
        },
      },
      {
        path: "profile",
        name: "Profile",
        component: Profile,
        meta: {
          requiresAgent: true,
        },
      },
      {
        path: "more",
        name: "More",
        component: More,
        meta: {
          requiresAgent: true,
        },
      },
      {
        path: "knowledge",
        name: "Knowledge",
        component: Knowledge,
        meta: {
          requiresAgent: true,
        },
      },
    ],
  },
  {
    path: "/:pathMatch(.*)*",
    redirect: "/",
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

const getDefaultPath = (store) => {
  const role = store.user?.role;
  return role === "admin" ? "/home/dashboard" : "/home/inbox";
};

router.beforeEach((to, from, next) => {
  const store = useStore();
  const requiresAuth = to.matched.some((record) => record.meta.requiresAuth !== false);
  const requiresAdmin = to.matched.some((record) => record.meta.requiresAdmin);
  const requiresAgent = to.matched.some((record) => record.meta.requiresAgent);

  if (requiresAuth && !store.token) {
    next("/login");
  } else if (to.path === "/login" && store.token) {
    next(getDefaultPath(store));
  } else if (requiresAdmin && store.user?.role !== "admin") {
    next(getDefaultPath(store));
  } else if (
    requiresAgent &&
    store.user?.role !== "agent" &&
    store.user?.role !== "admin"
  ) {
    next(getDefaultPath(store));
  } else if ((to.path === "/home" || to.path === "/home/") && store.token) {
    next(getDefaultPath(store));
  } else {
    next();
  }
});

export default router;
