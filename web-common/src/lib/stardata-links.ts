/**
 * StarData 对外链接统一配置。
 *
 * 文档站地址默认使用占位域名，正式域名确定后可通过环境变量
 * `VITE_STARDATA_DOCS_URL` 覆盖（或直接修改此处默认值）。
 */
export const DOCS_BASE_URL: string =
  (import.meta.env?.VITE_STARDATA_DOCS_URL as string | undefined) ??
  "https://docs.stardata.local";

/** 拼接文档站内页地址，path 需以 "/" 开头，如 docsUrl("/build/models") */
export function docsUrl(path = ""): string {
  return `${DOCS_BASE_URL}${path}`;
}

/**
 * 社区/支持链接。私有化部署无公开社区，置空表示隐藏相关入口；
 * 如需开放，填入支持页或工单系统地址即可。
 */
export const COMMUNITY_URL: string = "";
