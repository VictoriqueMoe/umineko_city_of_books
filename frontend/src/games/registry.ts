import type { GameType } from "../types/api";

export interface GameTypeDefinition {
    type: GameType;
    label: string;
    tagline: string;
    hubPath: string;
    newPath: string;
    detailPath: (id: string) => string;
    available: boolean;
    howToPlay?: string[];
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
        howToPlay: [
            "Click Start a new chess game, pick a player by username or from your mutual followers and send the invite. Your opponent plays as black; you play as white.",
            "Once they accept, drag a piece to a legal square to move. Illegal moves are rejected. You'll get a notification when it's your turn.",
            "Games are correspondence-style with no clocks, so take as long as you need between moves. The board updates live as soon as your opponent moves.",
            "If either player disconnects during an active game, they have 60 seconds to reconnect before they forfeit the match.",
            "Active games are public: anyone can open your board and watch. Spectators have their own side chat that players can't see. Finished games stay archived and browsable by everyone under Past Games.",
        ],
    },
];

export function gameTypeLabel(type: string): string {
    const hit = GAME_TYPES.find(g => g.type === type);
    return hit ? hit.label : type;
}

export function gameTypeFor(type: string): GameTypeDefinition | undefined {
    return GAME_TYPES.find(g => g.type === type);
}
