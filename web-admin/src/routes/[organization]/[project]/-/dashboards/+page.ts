import { redirect } from "@sveltejs/kit";

// StarData: the legacy dashboards listing is superseded by the portal boards
// page. Redirect so old links and bookmarks keep working.
export const load = ({ params: { organization, project }, url }) => {
  throw redirect(307, `/${organization}/${project}/boards${url.search}`);
};
