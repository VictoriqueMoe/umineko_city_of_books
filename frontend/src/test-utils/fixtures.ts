import type { SiteInfo } from "../api/endpoints";
import type { AuthContextValue } from "../context/authContextValue";
import type { GifFavouritesContextValue } from "../context/gifFavouritesContextValue";
import type { NotificationContextValue } from "../context/notificationContextValue";
import type { ThemeContextValue } from "../context/themeContextValue";
import type { UserProfile, UserStats } from "../types/api";

export function makeStats(overrides: Partial<UserStats> = {}): UserStats {
    return {
        theory_count: 0,
        response_count: 0,
        votes_received: 0,
        ship_count: 0,
        mystery_count: 0,
        fanfic_count: 0,
        ...overrides,
    };
}

export function makeUser(overrides: Partial<UserProfile> = {}): UserProfile {
    return {
        id: "00000000-0000-0000-0000-000000000001",
        username: "beatrice",
        display_name: "Beatrice",
        bio: "",
        avatar_url: "",
        banner_url: "",
        banner_position: 50,
        favourite_character: "",
        gender: "",
        pronoun_subject: "they",
        pronoun_possessive: "their",
        online: false,
        social_twitter: "",
        social_discord: "",
        social_waifulist: "",
        social_tumblr: "",
        social_github: "",
        social_bluesky: "",
        website: "",
        dms_enabled: true,
        episode_progress: 0,
        higurashi_arc_progress: 0,
        ciconia_chapter_progress: 0,
        secrets: [],
        created_at: "2026-01-01T00:00:00Z",
        stats: makeStats(),
        ...overrides,
    };
}

export function makeSiteInfo(overrides: Partial<SiteInfo> = {}): SiteInfo {
    return {
        site_name: "When They Cry",
        site_description: "",
        registration_type: "open",
        announcement_banner: "",
        default_theme: "featherine",
        maintenance_mode: false,
        maintenance_title: "",
        maintenance_message: "",
        turnstile_enabled: false,
        turnstile_site_key: "",
        voice_enabled: true,
        email_enabled: true,
        chatbot_enabled: false,
        chatbot_require_permission: false,
        max_image_size: 10 * 1024 * 1024,
        max_video_size: 50 * 1024 * 1024,
        max_audio_size: 25 * 1024 * 1024,
        new_account_hours: 24,
        top_detective_ids: [],
        top_gm_ids: [],
        top_chess_ids: [],
        top_checkers_ids: [],
        top_othello_ids: [],
        top_minesweeper_ids: [],
        vanity_roles: [],
        vanity_role_assignments: {},
        listed_secrets: [],
        rules_page: "",
        version: "test",
        app_latest_version: "",
        app_download_url: "",
        ...overrides,
    };
}

export function makeAuthContext(overrides: Partial<AuthContextValue> = {}): AuthContextValue {
    return {
        user: null,
        loading: false,
        setUser: () => {},
        loginUser: () => Promise.resolve(),
        registerUser: () => Promise.resolve(),
        logoutUser: () => Promise.resolve(),
        ...overrides,
    };
}

export function makeNotificationContext(overrides: Partial<NotificationContextValue> = {}): NotificationContextValue {
    return {
        unreadCount: 0,
        chatUnreadCount: 0,
        liveGamesCount: 0,
        liveStreamsCount: 0,
        markRead: () => Promise.resolve(),
        markAllRead: () => Promise.resolve(),
        addWSListener: () => () => {},
        sendWSMessage: () => {},
        wsEpoch: 0,
        ...overrides,
    };
}

export function makeThemeContext(overrides: Partial<ThemeContextValue> = {}): ThemeContextValue {
    return {
        theme: "featherine",
        setTheme: () => {},
        font: "default",
        setFont: () => {},
        wideLayout: false,
        setWideLayout: () => {},
        particlesEnabled: true,
        setParticlesEnabled: () => {},
        hasSecret: () => false,
        addSecret: () => {},
        ...overrides,
    };
}

export function makeGifFavouritesContext(
    overrides: Partial<GifFavouritesContextValue> = {},
): GifFavouritesContextValue {
    return {
        favourites: [],
        ids: new Set<string>(),
        isFavourite: () => false,
        toggle: () => Promise.resolve(),
        refresh: () => Promise.resolve(),
        ...overrides,
    };
}
