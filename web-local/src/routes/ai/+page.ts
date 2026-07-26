import { redirect } from "@sveltejs/kit";

export function load({ url }) {
  throw redirect(301, `/chat${url.search}`);
}
