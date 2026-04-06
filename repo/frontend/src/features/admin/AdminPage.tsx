import { useEffect, useMemo, useState } from "react";

import {
  assignUserRole,
  getRolePermissionMatrix,
  listTenantSettings,
  listUserRoleAssignments,
  revokeUserRole,
  saveRolePermissionMatrix,
  saveTenantSettings,
  type RolePermissionAssignment,
  type TenantSettings,
  type UserRoleAssignment,
} from "../../api/endpoints";
import { AccessGate } from "../../auth/access-control";
import { roles, type Role } from "../../auth/roles";
import { useSessionStore } from "../../state/session-store";

const editablePermissions = [
  "tenant.settings.view",
  "tenant.settings.manage",
  "rbac.matrix.view",
  "rbac.matrix.manage",
  "rbac.assignments.view",
  "rbac.assignments.manage",
];

export function AdminPage() {
  const role = useSessionStore((s) => s.user?.primaryRole ?? null);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);

  const [tenantSettings, setTenantSettings] = useState<TenantSettings | null>(
    null,
  );
  const [matrix, setMatrix] = useState<RolePermissionAssignment[]>([]);
  const [userRoles, setUserRoles] = useState<UserRoleAssignment[]>([]);

  const [assignUserID, setAssignUserID] = useState("");
  const [assignRole, setAssignRole] = useState<Role>("learner");

  const matrixIndex = useMemo(() => {
    const map = new Map<string, boolean>();
    for (const item of matrix) {
      map.set(`${item.role}:${item.permission}`, item.allowed);
    }
    return map;
  }, [matrix]);

  const loadAll = async () => {
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      const [settingsList, permissionRows, userRoleRows] = await Promise.all([
        listTenantSettings(),
        getRolePermissionMatrix(),
        listUserRoleAssignments(),
      ]);
      setTenantSettings(settingsList[0] ?? null);
      setMatrix(permissionRows);
      setUserRoles(userRoleRows);
      setStatus("Administrator data loaded");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAll();
  }, []);

  const togglePermission = (currentRole: Role, permission: string) => {
    setMatrix((prev) => {
      const key = `${currentRole}:${permission}`;
      const nextAllowed = !matrixIndex.get(key);
      const next = prev.filter(
        (item) => !(item.role === currentRole && item.permission === permission),
      );
      next.push({ role: currentRole, permission, allowed: nextAllowed });
      return next;
    });
  };

  const saveSettings = async () => {
    if (!tenantSettings) {
      setError("Tenant settings not available");
      return;
    }
    if (!tenantSettings.tenant_slug.trim() || !tenantSettings.tenant_name.trim()) {
      setError("Tenant name and slug are required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await saveTenantSettings(tenantSettings);
      setStatus("Tenant settings saved");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const saveMatrix = async () => {
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await saveRolePermissionMatrix(matrix);
      setStatus("Role-permission matrix updated");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const onAssignRole = async () => {
    if (!assignUserID.trim()) {
      setError("User ID is required");
      return;
    }
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await assignUserRole(assignUserID.trim(), assignRole);
      await loadAll();
      setStatus("Role assigned");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  const onRevokeRole = async (userID: string, userRole: Role) => {
    setLoading(true);
    setError(null);
    setStatus(null);
    try {
      await revokeUserRole(userID, userRole);
      await loadAll();
      setStatus("Role revoked");
    } catch (e) {
      setError(messageFromError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <section>
      <h2>Administrator</h2>
      <p>Manage tenant settings, role permissions, and role assignments.</p>

      {error ? <p className="error">{error}</p> : null}
      {status ? <p>{status}</p> : null}

      <AccessGate
        role={role}
        permission="admin.manage"
        fallback={<p>Administrator privileges are required.</p>}
      >
        <div className="login-panel">
          <h3>Tenant Settings</h3>
          <div className="login-row">
            <input
              value={tenantSettings?.tenant_name ?? ""}
              onChange={(e) =>
                setTenantSettings((prev) =>
                  prev
                    ? {
                        ...prev,
                        tenant_name: e.target.value,
                      }
                    : prev,
                )
              }
              placeholder="tenant name"
            />
            <input
              value={tenantSettings?.tenant_slug ?? ""}
              onChange={(e) =>
                setTenantSettings((prev) =>
                  prev
                    ? {
                        ...prev,
                        tenant_slug: e.target.value,
                      }
                    : prev,
                )
              }
              placeholder="tenant slug"
            />
            <input
              type="number"
              min={1}
              max={20}
              value={tenantSettings?.max_active_bookings_per_learner ?? 3}
              onChange={(e) =>
                setTenantSettings((prev) =>
                  prev
                    ? {
                        ...prev,
                        max_active_bookings_per_learner: Number(e.target.value),
                      }
                    : prev,
                )
              }
              placeholder="max bookings"
            />
            <button onClick={saveSettings} disabled={loading || !tenantSettings}>
              {loading ? "Saving..." : "Save Settings"}
            </button>
          </div>
          <div className="login-row">
            <label>
              <input
                type="checkbox"
                checked={tenantSettings?.allow_self_registration ?? false}
                onChange={(e) =>
                  setTenantSettings((prev) =>
                    prev
                      ? {
                          ...prev,
                          allow_self_registration: e.target.checked,
                        }
                      : prev,
                  )
                }
              />
              Allow self registration
            </label>
            <label>
              <input
                type="checkbox"
                checked={tenantSettings?.require_mfa ?? false}
                onChange={(e) =>
                  setTenantSettings((prev) =>
                    prev
                      ? {
                          ...prev,
                          require_mfa: e.target.checked,
                        }
                      : prev,
                  )
                }
              />
              Require MFA
            </label>
          </div>
        </div>

        <div className="login-panel">
          <h3>Role-Permission Matrix</h3>
          <table>
            <thead>
              <tr>
                <th>Permission</th>
                {roles.map((item) => (
                  <th key={item}>{item}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {editablePermissions.map((permission) => (
                <tr key={permission}>
                  <td>{permission}</td>
                  {roles.map((item) => {
                    const key = `${item}:${permission}`;
                    const checked = matrixIndex.get(key) ?? false;
                    return (
                      <td key={key}>
                        <input
                          aria-label={`${item}:${permission}`}
                          type="checkbox"
                          checked={checked}
                          onChange={() => togglePermission(item, permission)}
                        />
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
          <button onClick={saveMatrix} disabled={loading || matrix.length === 0}>
            {loading ? "Saving..." : "Save Matrix"}
          </button>
        </div>

        <div className="login-panel">
          <h3>User Role Assignments</h3>
          <div className="login-row">
            <input
              value={assignUserID}
              onChange={(e) => setAssignUserID(e.target.value)}
              placeholder="user id"
            />
            <select
              value={assignRole}
              onChange={(e) => setAssignRole(e.target.value as Role)}
            >
              {roles.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
            <button onClick={onAssignRole} disabled={loading}>
              {loading ? "Assigning..." : "Assign Role"}
            </button>
          </div>
          <ul>
            {userRoles.map((item) => (
              <li key={item.user_id}>
                {item.username} ({item.user_id}) - {item.roles.join(", ") || "no roles"}
                {item.roles.map((userRole) => (
                  <button
                    key={`${item.user_id}-${userRole}`}
                    onClick={() => onRevokeRole(item.user_id, userRole)}
                    disabled={loading}
                  >
                    Revoke {userRole}
                  </button>
                ))}
              </li>
            ))}
          </ul>
        </div>
      </AccessGate>
    </section>
  );
}

function messageFromError(e: unknown): string {
  if (typeof e === "object" && e && "message" in e) {
    return String((e as { message: string }).message);
  }
  return "Request failed";
}
