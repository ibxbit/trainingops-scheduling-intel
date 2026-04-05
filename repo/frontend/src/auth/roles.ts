export const roles = [
  "administrator",
  "program_coordinator",
  "instructor",
  "learner",
] as const;

export type Role = (typeof roles)[number];

export type SessionUser = {
  userId: string;
  tenantId: string;
  roles: Role[];
  primaryRole: Role;
};

export const roleLabels: Record<Role, string> = {
  administrator: "Administrator",
  program_coordinator: "Program Coordinator",
  instructor: "Instructor",
  learner: "Learner",
};

const rolePriority: Role[] = [
  "administrator",
  "program_coordinator",
  "instructor",
  "learner",
];

export function normalizeRoles(rawRoles: string[]): Role[] {
  const deduped: Role[] = [];
  for (const role of rawRoles) {
    if (roles.includes(role as Role) && !deduped.includes(role as Role)) {
      deduped.push(role as Role);
    }
  }
  return deduped;
}

export function derivePrimaryRole(rawRoles: string[]): Role {
  const normalized = normalizeRoles(rawRoles);
  for (const role of rolePriority) {
    if (normalized.includes(role)) {
      return role;
    }
  }
  return "learner";
}
