import type { Role } from "./roles";

export type Permission =
  | "calendar.view"
  | "calendar.manage"
  | "booking.view"
  | "booking.manage"
  | "content.view"
  | "content.manage"
  | "planning.view"
  | "planning.manage"
  | "dashboard.view"
  | "dashboard.manage";

export const routePolicies: Record<string, Permission> = {
  "/dashboard": "dashboard.view",
  "/calendar": "calendar.view",
  "/booking": "booking.view",
  "/content": "content.view",
  "/planning": "planning.view",
};

const policyMatrix: Record<Permission, Role[]> = {
  "dashboard.view": [
    "administrator",
    "program_coordinator",
    "instructor",
    "learner",
  ],
  "dashboard.manage": ["administrator", "program_coordinator"],
  "calendar.view": [
    "administrator",
    "program_coordinator",
    "instructor",
    "learner",
  ],
  "calendar.manage": ["administrator", "program_coordinator"],
  "booking.view": [
    "administrator",
    "program_coordinator",
    "instructor",
    "learner",
  ],
  "booking.manage": ["administrator", "program_coordinator", "learner"],
  "content.view": [
    "administrator",
    "program_coordinator",
    "instructor",
    "learner",
  ],
  "content.manage": ["administrator", "program_coordinator"],
  "planning.view": [
    "administrator",
    "program_coordinator",
    "instructor",
    "learner",
  ],
  "planning.manage": ["administrator", "program_coordinator", "instructor"],
};

export function canAccess(role: Role | null, permission: Permission): boolean {
  if (!role) {
    return false;
  }
  return policyMatrix[permission].includes(role);
}
