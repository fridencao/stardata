import { redirect } from "@sveltejs/kit";

// StarData: the raw Rill dashboards page is superseded by the business
// portal's Boards page (`/boards`). Redirect legacy bookmarks/deeplinks.
export const load = ({ params: { organization, project } }) => {
  throw redirect(307, `/${organization}/${project}/boards`);
};
