/** 业务门户专属路由前缀("/" 需精确匹配,见 isPortalRoute) */
export const PORTAL_ROUTE_PREFIXES = ["/chat", "/boards"] as const;

/** 高级模式(旧 IDE)路由前缀,使用统一的 PortalNav + StudioTabs 壳 */
export const IDE_ROUTE_PREFIXES = ["/files", "/connector/", "/graph"] as const;

/** 技术侧(Studio + 旧 IDE)专属路由前缀 */
export const STUDIO_ROUTE_PREFIXES = [
  "/studio",
  ...IDE_ROUTE_PREFIXES,
  "/status",
] as const;

/** 双侧共享路由(不触发模式切换) */
const SHARED_PREFIXES = [
  "/welcome",
  "/explore/",
  "/canvas/",
  "/-/",
] as const;

/** 后端 --preview 锁定模式下允许的全部前缀 */
export const PORTAL_ALLOWED_PREFIXES = [
  ...PORTAL_ROUTE_PREFIXES,
  ...SHARED_PREFIXES,
] as const;

export function isPortalRoute(pathname: string): boolean {
  return (
    pathname === "/" ||
    PORTAL_ROUTE_PREFIXES.some((prefix) => pathname.startsWith(prefix))
  );
}

export function isStudioRoute(pathname: string): boolean {
  return STUDIO_ROUTE_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}

export function isIdeRoute(pathname: string): boolean {
  return IDE_ROUTE_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}
