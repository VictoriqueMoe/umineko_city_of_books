import type { Announcement, Art, Gallery, Post, SiteInfoSecret } from "../../types/api";
import { makePublicUser } from "./user";

export function makePost(overrides: Partial<Post> = {}): Post {
    return {
        id: "post-1",
        author: makePublicUser(),
        body: "Without love it cannot be seen",
        media: [],
        share_count: 0,
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

export function makeArt(overrides: Partial<Art> = {}): Art {
    return {
        id: "art-1",
        author: makePublicUser(),
        corner: "general",
        art_type: "drawing",
        title: "Golden Butterflies",
        description: "",
        image_url: "/art-1-full.png",
        thumbnail_url: "/art-1-thumb.png",
        tags: [],
        like_count: 0,
        comment_count: 0,
        view_count: 0,
        user_liked: false,
        is_spoiler: false,
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

export function makeGallery(overrides: Partial<Gallery> = {}): Gallery {
    return {
        id: "gallery-1",
        author: makePublicUser(),
        name: "Golden Butterflies",
        description: "",
        cover_image_url: "",
        cover_thumbnail_url: "",
        art_count: 3,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

export function makeAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
    return {
        id: "announcement-1",
        title: "The board reopens",
        body: "The game board is open again.",
        author: makePublicUser(),
        pinned: false,
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

export function makeSiteSecret(overrides: Partial<SiteInfoSecret> = {}): SiteInfoSecret {
    return {
        id: "epitaph",
        title: "The Witch's Epitaph",
        description: "Seek the key that opens the golden land.",
        solved: false,
        pieces: [],
        ...overrides,
    };
}
