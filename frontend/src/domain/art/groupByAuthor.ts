import type { Gallery, User } from "../../types/api";

export interface ArtistGalleries {
    user: User;
    galleries: Gallery[];
}

export function groupByAuthor(galleries: Gallery[]): ArtistGalleries[] {
    const grouped = new Map<string, ArtistGalleries>();

    for (const gallery of galleries) {
        const existing = grouped.get(gallery.author.id);
        if (existing) {
            existing.galleries.push(gallery);
            continue;
        }

        grouped.set(gallery.author.id, { user: gallery.author, galleries: [gallery] });
    }

    return [...grouped.values()].sort((a, b) => a.user.display_name.localeCompare(b.user.display_name));
}
