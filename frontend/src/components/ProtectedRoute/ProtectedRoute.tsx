import { Navigate, Outlet, useLocation } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import type { Permission } from "../../utils/permissions";
import { can } from "../../utils/permissions";

interface ProtectedRouteProps {
    permission?: Permission;
}

export function ProtectedRoute({ permission }: ProtectedRouteProps) {
    const { user, loading } = useAuth();
    const originalLocation = useLocation();

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    if (!user) {
        return <Navigate to="/login" state={{ from: originalLocation }} replace />;
    }

    if (permission && !can(user.role, permission)) {
        return <Navigate to="/" replace />;
    }

    return <Outlet />;
}
