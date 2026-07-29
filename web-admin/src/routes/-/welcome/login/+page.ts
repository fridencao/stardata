import { redirect } from "@sveltejs/kit";
import type { PageData } from "./$types";

export const load = async ({ parent, url }) => {
  const { user } = await parent();

  // Already authenticated: send the user back to the originally requested
  // page (or home). The `redirect` query param is set by redirectToLogin().
  if (user) {
    const redirectTo = url.searchParams.get("redirect");
    throw redirect(307, redirectTo ?? "/");
  }

  return {};
};
