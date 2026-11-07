import { Navigate, Outlet } from "react-router";
import { type EditableResource, useEditable } from "../../hooks/useEditable";

type EditableRouteProps = {
    resource: EditableResource;
};

export function EditableRoute({ resource }: EditableRouteProps) {
    const { loading, allowed } = useEditable(resource);

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    if (!allowed) {
        return <Navigate to="/" replace />;
    }

    return <Outlet />;
}
