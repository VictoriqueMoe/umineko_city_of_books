import type { PaginationFields } from "./common";

export interface VanityRoleUsersResponse extends PaginationFields {
    users: { id: string; username: string; display_name: string; avatar_url: string }[];
}

export interface PermissionCatalogueItem {
    permission: string;
    label: string;
    vanity_assignable: boolean;
}

export interface RolePermissionsItem {
    role: string;
    label: string;
    permissions: string[];
}

export interface VanityRolePermissionsItem {
    id: string;
    label: string;
    color: string;
    sort_order: number;
    permissions: string[];
}

export interface PermissionSettingsResponse {
    permissions: PermissionCatalogueItem[];
    roles: RolePermissionsItem[];
    vanity_roles: VanityRolePermissionsItem[];
}
