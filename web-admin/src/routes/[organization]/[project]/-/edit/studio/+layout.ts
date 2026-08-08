import { redirect } from "@sveltejs/kit";

// StarData: Studio moved to a top-level route (`/studio/[domain]`), decoupling
// it from the `-/edit` dev-deployment lifecycle (see
// design/phase4-enterprise-app.md §3.1 and phase4-review-and-hardening.md D10).
// Permanently redirect old bookmarks; `project` is the new `[domain]`, and any
// sub-path (`/sources`, `/publish`, ...) is preserved so deep links survive.
export const load = async ({ params: { project }, url }) => {
  const prefix = url.pathname.indexOf("/-/edit/studio");
  const subpath =
    prefix >= 0 ? url.pathname.slice(prefix + "/-/edit/studio".length) : "";
  throw redirect(308, `/studio/${project}${subpath}${url.search}`);
};
