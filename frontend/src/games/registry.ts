import type { GameType } from "../types/api";

export interface GameTypeDefinition {
    type: GameType;
    label: string;
    tagline: string;
    hubPath: string;
    newPath: string;
    detailPath: (id: string) => string;
    available: boolean;
}

export const GAME_TYPES: GameTypeDefinition[] = [
    {
        type: "chess",
        label: "Chess",
        tagline: "Correspondence-style matches against other players. Invite someone to a board.",
        hubPath: "/games/chess",
        newPath: "/games/chess/new",
        detailPath: (id: string) => `/games/chess/${id}`,
        available: true,
    },
];

export function gameTypeLabel(type: string): string {
    const hit = GAME_TYPES.find(g => g.type === type);
    return hit ? hit.label : type;
}

export function gameTypeFor(type: string): GameTypeDefinition | undefined {
    return GAME_TYPES.find(g => g.type === type);
}
