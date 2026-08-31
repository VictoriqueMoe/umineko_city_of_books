import type { SessionUser, User, UserStats } from "../../types/api";

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

export function makeUser(overrides: Partial<SessionUser> = {}): SessionUser {
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

export function makePublicUser(overrides: Partial<User> = {}): User {
    return {
        id: "u1",
        username: "beatrice",
        display_name: "Beatrice",
        ...overrides,
    };
}
