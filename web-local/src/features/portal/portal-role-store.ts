import { browser } from "$app/environment";
import { writable } from "svelte/store";

export type PortalRole = "business" | "tech";

const STORAGE_KEY = "stardata-portal-role";

function createPortalRoleStore() {
  // 默认 tech:本地开发者/实施人员首次进入可见完整能力
  const initial: PortalRole =
    browser && localStorage.getItem(STORAGE_KEY) === "business"
      ? "business"
      : "tech";
  const { subscribe, set } = writable<PortalRole>(initial);
  return {
    subscribe,
    set(role: PortalRole) {
      if (browser) localStorage.setItem(STORAGE_KEY, role);
      set(role);
    },
  };
}

export const portalRole = createPortalRoleStore();
