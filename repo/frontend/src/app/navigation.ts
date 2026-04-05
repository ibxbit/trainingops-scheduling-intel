import type { Role } from "../auth/roles";
import { canAccess } from "../auth/policy";
import { appRoutes } from "./route-config";

export type NavItem = {
  key: string;
  label: string;
  path: string;
};

export function navigationForRole(role: Role): NavItem[] {
  return appRoutes
    .filter((r) => canAccess(role, r.permission))
    .map((r) => ({
      key: r.key,
      label: r.title,
      path: r.path,
    }));
}
