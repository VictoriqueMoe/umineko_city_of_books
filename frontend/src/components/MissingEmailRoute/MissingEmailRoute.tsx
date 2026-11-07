import { Navigate, Outlet } from "react-router";
import { useAuthedUser } from "../../hooks/useAuthedUser";

export function MissingEmailRoute() {
    const user = useAuthedUser();

    if (user.email) {
        return <Navigate to="/" replace />;
    }

    return <Outlet />;
}
