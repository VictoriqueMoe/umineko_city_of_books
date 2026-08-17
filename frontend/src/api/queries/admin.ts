import { useQuery } from "@tanstack/react-query";
import {
    getAdminSettings,
    getAdminStats,
    getAdminUser,
    getAdminUsers,
    getAuditLog,
    getBannedGifs,
    getChatbotBasePrompts,
    getChatbotModels,
    getChatbotUsage,
    getChatbots,
    getInvites,
    getReports,
    getUserAuditLog,
    getUserIPMatches,
    getVanityRoleUsers,
    getAdminPermissions,
    getVanityRoles,
    listAnnouncements,
    listGlobalBannedWords,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useAdminAnnouncements() {
    const query = useQuery({
        queryKey: queryKeys.admin.announcements(),
        queryFn: () => listAnnouncements(100, 0),
    });
    return {
        announcements: query.data?.announcements ?? [],
        loading: query.isLoading,
        refresh: query.refetch,
    };
}

export function useAdminUsers(search: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.admin.users({ search, limit, offset }),
        queryFn: () => getAdminUsers({ search, limit, offset }),
    });
    return {
        users: query.data?.users ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
        error: query.isError,
        refresh: query.refetch,
    };
}

export function useAdminUser(id: string) {
    const query = useQuery({
        queryKey: queryKeys.admin.userDetail(id),
        queryFn: () => getAdminUser(id),
        enabled: !!id,
    });
    return { user: query.data ?? null, loading: query.isLoading };
}

export function useUserIPMatches(id: string, enabled: boolean) {
    const query = useQuery({
        queryKey: queryKeys.admin.userIpMatches(id),
        queryFn: () => getUserIPMatches(id),
        enabled: !!id && enabled,
    });
    return {
        ip: query.data?.ip ?? "",
        users: query.data?.users ?? [],
        loading: query.isLoading,
        failed: query.isError,
    };
}

export function useUserAuditLog(id: string, enabled: boolean, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.admin.userAuditLog(id, limit, offset),
        queryFn: () => getUserAuditLog(id, limit, offset),
        enabled: !!id && enabled,
    });
    return {
        entries: query.data?.entries ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
        failed: query.isError,
    };
}

export function useAdminStats() {
    const query = useQuery({
        queryKey: queryKeys.admin.stats(),
        queryFn: () => getAdminStats(),
    });
    return { stats: query.data ?? null, loading: query.isLoading };
}

export function useAdminSettings() {
    const query = useQuery({
        queryKey: queryKeys.admin.settings(),
        queryFn: () => getAdminSettings(),
    });
    return { settings: query.data ?? null, loading: query.isLoading, refresh: query.refetch };
}

export function useAuditLog(action: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.admin.auditLog({ action, limit, offset }),
        queryFn: () => getAuditLog({ action: action || undefined, limit, offset }),
    });
    return {
        entries: query.data?.entries ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}

export function useInvites(limit: number, offset: number) {
    const query = useQuery({
        queryKey: [...queryKeys.admin.invites(), limit, offset],
        queryFn: () => getInvites({ limit, offset }),
    });
    return { invites: query.data?.invites ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useReports(status: string) {
    const query = useQuery({
        queryKey: queryKeys.admin.reports({ status }),
        queryFn: () => getReports(status),
    });
    return { reports: query.data?.reports ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useBannedGifs() {
    const query = useQuery({
        queryKey: queryKeys.admin.bannedGifs(),
        queryFn: () => getBannedGifs(),
    });
    return { entries: query.data?.entries ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useGlobalBannedWords() {
    const query = useQuery({
        queryKey: queryKeys.admin.bannedWords("global"),
        queryFn: () => listGlobalBannedWords(),
    });
    return { rules: query.data?.rules ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useVanityRoles() {
    const query = useQuery({
        queryKey: queryKeys.admin.vanityRoles(),
        queryFn: () => getVanityRoles(),
    });
    return { roles: query.data ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useAdminPermissions(enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.admin.permissions(),
        queryFn: () => getAdminPermissions(),
        enabled,
    });
    return {
        catalogue: query.data?.permissions ?? [],
        roles: query.data?.roles ?? [],
        vanityRoles: query.data?.vanity_roles ?? [],
        loading: enabled && query.isLoading,
        refresh: query.refetch,
    };
}

export function useChatbots() {
    const query = useQuery({
        queryKey: queryKeys.admin.chatbots(),
        queryFn: () => getChatbots(),
    });
    return { bots: query.data ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useChatbotBasePrompts() {
    const query = useQuery({
        queryKey: queryKeys.admin.chatbotBasePrompts(),
        queryFn: () => getChatbotBasePrompts(),
    });
    return { basePrompts: query.data?.base_prompts ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useChatbotUsage(days: number) {
    const query = useQuery({
        queryKey: queryKeys.admin.chatbotUsage(days),
        queryFn: () => getChatbotUsage(days),
    });
    return { usage: query.data ?? null, loading: query.isLoading, refresh: query.refetch };
}

export function useChatbotModels() {
    const query = useQuery({
        queryKey: queryKeys.admin.chatbotModels(),
        queryFn: () => getChatbotModels(),
        staleTime: 5 * 60_000,
    });

    function refresh() {
        query.refetch().catch(() => undefined);
    }

    return {
        models: query.data?.models ?? [],
        modelsError: query.data?.error ?? "",
        loading: query.isLoading,
        refresh,
    };
}

export function useVanityRoleUsers(id: string, search: string, limit: number, offset: number) {
    const query = useQuery({
        queryKey: queryKeys.admin.vanityRoleUserList(id, search, limit, offset),
        queryFn: () => getVanityRoleUsers(id, { search: search || undefined, limit, offset }),
        enabled: !!id,
    });
    return {
        users: query.data?.users ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}
