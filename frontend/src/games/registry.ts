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
    {
        type: "checkers",
        label: "Checkers",
        tagline: "Classic American draughts. Jump your opponent's pieces, crown your kings.",
        hubPath: "/games/checkers",
        newPath: "/games/checkers/new",
        detailPath: (id: string) => `/games/checkers/${id}`,
        available: true,
        howToPlay: [
            "Click Start a new checkers game, pick a player by username or from your mutual followers and send the invite. Your opponent plays black; you play red.",
            "Red moves first. Men move one diagonal step forward onto an empty dark square. If a capture (jump) is available anywhere on the board, you must take it.",
            "Jump an adjacent opponent piece by landing on the empty dark square beyond it. If more jumps chain from your landing square, you must keep jumping in the same turn.",
            "Reaching the far rank crowns your man into a king, which can move and jump both forwards and backwards. A man that crowns mid-jump stops for that turn.",
            "You win by capturing all your opponent's pieces or leaving them with no legal move. If 40 turns pass with no captures, the game is drawn.",
            "Games are correspondence-style with no clocks. Disconnects trigger a 60-second forfeit timer. Active games are public to spectators; finished games are archived under Past Games.",
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
