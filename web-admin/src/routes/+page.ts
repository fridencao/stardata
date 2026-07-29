import { redirectToLogin } from "@rilldata/web-admin/client/redirect-utils";

export async function load({ parent, url }) {
  const { user } = await parent();

  if (!user) redirectToLogin(url.pathname + url.search);

  return { user };
}
