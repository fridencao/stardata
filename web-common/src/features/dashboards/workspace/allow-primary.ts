import { writable } from "svelte/store";

/**
 * 原 DeployProjectCTA 的共享状态。Deploy 入口已随私有化部署移除，
 * 保留该 store 供工作区按钮（CreateDashboardButton / GoToDashboardButton）
 * 决定主/次按钮样式，默认 false 即始终以主按钮展示。
 */
export const allowPrimary = writable(false);
