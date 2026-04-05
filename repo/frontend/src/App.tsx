import {
  type ChangeEvent,
  type ReactElement,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";

import { login, logout, me } from "./api/endpoints";
import { appRoutes } from "./app/route-config";
import { navigationForRole } from "./app/navigation";
import { AccessGate } from "./auth/access-control";
import {
  derivePrimaryRole,
  normalizeRoles,
  roleLabels,
  type Role,
} from "./auth/roles";
import { useSessionStore } from "./state/session-store";

function Forbidden() {
  return <div className="error">You do not have access to this page.</div>;
}

function RequireAuth({ children }: { children: ReactElement }) {
  const user = useSessionStore((s) => s.user);
  const isReady = useSessionStore((s) => s.isReady);
  const location = useLocation();
  if (!isReady) {
    return <p>Loading session...</p>;
  }
  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }
  return children;
}

export function App() {
  const user = useSessionStore((s) => s.user);
  const setUser = useSessionStore((s) => s.setUser);
  const setReady = useSessionStore((s) => s.setReady);
  const clearSession = useSessionStore((s) => s.clearSession);

  const [tenantSlug, setTenantSlug] = useState("acme-training");
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("AdminPass1234");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    let active = true;
    const boot = async () => {
      try {
        const profile = await me();
        if (!active) {
          return;
        }
        const roles = normalizeRoles(profile.roles);
        const primaryRole = derivePrimaryRole(profile.roles);
        setUser({
          userId: profile.user_id,
          tenantId: profile.tenant_id,
          roles,
          primaryRole,
        });
      } catch {
        if (!active) {
          return;
        }
        clearSession();
      } finally {
        if (active) {
          setReady(true);
        }
      }
    };
    boot();
    return () => {
      active = false;
    };
  }, [clearSession, setReady, setUser]);

  const role: Role | null = user?.primaryRole ?? null;
  const nav = useMemo(() => (role ? navigationForRole(role) : []), [role]);

  const onLogin = async () => {
    if (loading) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await login({ tenant_slug: tenantSlug, username, password });
      const profile = await me();
      const roles = normalizeRoles(profile.roles);
      const primaryRole = derivePrimaryRole(profile.roles);
      setUser({
        userId: profile.user_id,
        tenantId: profile.tenant_id,
        roles,
        primaryRole,
      });
      setReady(true);
      navigate("/dashboard", { replace: true });
    } catch (e) {
      setError(
        typeof e === "object" && e && "message" in e
          ? String((e as { message: string }).message)
          : "Login failed",
      );
      clearSession();
      setReady(true);
    } finally {
      setLoading(false);
    }
  };

  const onLogout = async () => {
    if (loading) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await logout();
    } catch {
      // do nothing; local reset still required
    } finally {
      clearSession();
      setReady(true);
      setLoading(false);
      navigate("/login", { replace: true });
    }
  };

  if (location.pathname === "/login") {
    return (
      <div className="auth-shell">
        <section className="login-panel">
          <h2>Sign in</h2>
          <div className="login-row">
            <input
              value={tenantSlug}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setTenantSlug(e.target.value)
              }
              placeholder="tenant slug"
            />
            <input
              value={username}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setUsername(e.target.value)
              }
              placeholder="username"
            />
            <input
              value={password}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setPassword(e.target.value)
              }
              placeholder="password"
              type="password"
            />
            <button onClick={onLogin} disabled={loading}>
              {loading ? "Signing in..." : "Login"}
            </button>
          </div>
          {error ? <p className="error">{error}</p> : null}
        </section>
      </div>
    );
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <h1>TrainingOps</h1>
        <p>{user ? `${roleLabels[user.primaryRole]} session` : "Guest"}</p>
        <nav>
          {nav.map((item) => (
            <button
              key={item.key}
              className={location.pathname === item.path ? "active" : ""}
              onClick={() => navigate(item.path)}
            >
              {item.label}
            </button>
          ))}
        </nav>
        {user ? (
          <button
            className="logout-button"
            onClick={onLogout}
            disabled={loading}
          >
            Logout
          </button>
        ) : null}
      </aside>
      <main className="main-content">
        <Routes>
          <Route path="/login" element={<Navigate to="/dashboard" replace />} />
          {appRoutes.map((route) => (
            <Route
              key={route.path}
              path={route.path}
              element={
                <RequireAuth>
                  <AccessGate
                    role={role}
                    permission={route.permission}
                    fallback={<Forbidden />}
                  >
                    <route.component />
                  </AccessGate>
                </RequireAuth>
              }
            />
          ))}
          <Route
            path="*"
            element={<Navigate to={user ? "/dashboard" : "/login"} replace />}
          />
        </Routes>
      </main>
    </div>
  );
}
