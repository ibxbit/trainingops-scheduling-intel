import type { PropsWithChildren, ReactNode } from "react";

import type { Permission } from "./policy";
import { canAccess } from "./policy";
import type { Role } from "./roles";

type AccessGateProps = PropsWithChildren<{
  role: Role | null;
  permission: Permission;
  fallback?: ReactNode;
}>;

export function AccessGate({
  role,
  permission,
  fallback = null,
  children,
}: AccessGateProps) {
  if (!canAccess(role, permission)) {
    return <>{fallback}</>;
  }
  return <>{children}</>;
}
