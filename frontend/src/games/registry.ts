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
    {
        type: "othello",
        label: "Othello",
        tagline: "The modern Reversi. Place a disc to flank, then watch the colours flip.",
        hubPath: "/games/othello",
        newPath: "/games/othello/new",
        detailPath: (id: string) => `/games/othello/${id}`,
        available: true,
        howToPlay: [
            "Click Start a new othello game, pick a player by username or from your mutual followers and send the invite. Your opponent plays white; you play black.",
            "Black moves first. The four centre squares start with the standard cross: white on D4 and E5, black on E4 and D5.",
            "On your turn, place a disc on an empty square so that it flanks at least one straight line (orthogonal or diagonal) of your opponent's discs between the new disc and one of yours. Every flanked disc flips to your colour.",
            "If you have no legal placement, the server passes for you automatically and the turn returns to your opponent. The game ends when both sides have no legal moves in a row, or when the board fills up.",
            "Whoever holds the most discs at the end wins. An equal split is a draw. Corners are permanent once captured, so plan your edge play around them.",
            "Games are correspondence-style with no clocks. Disconnects trigger a 60-second forfeit timer. Active games are public to spectators; finished games are archived under Past Games.",
        ],
    },
    {
        type: "minesweeper",
        label: "Minesweeper",
        tagline: "Real-time duel on a shared minefield. Pick a character, race to clear the board.",
        hubPath: "/games/minesweeper",
        newPath: "/games/minesweeper/new",
        detailPath: (id: string) => `/games/minesweeper/${id}`,
        available: true,
        howToPlay: [
            "Click Start a new minesweeper game, pick a player and send the invite. Once they accept, you both pick a character from the Umineko cast: Bernkastel, Erika, Dlanor or Lambdadelta.",
            "After both characters are chosen, the match begins. You both play simultaneously on the same minefield, with independent reveal grids.",
            "Left-click a cell to reveal it. Right-click to toggle a flag on suspected mines. The first reveal is always safe; mines are placed lazily after both players' first clicks.",
            "Clear all safe cells before your opponent to win. Hitting a mine instantly hands the win to your opponent.",
            "If either player disconnects mid-game, a 60-second forfeit timer runs before the connected player wins automatically.",
            "Active games are public to spectators, who see both boards via mini-views. Mine positions stay hidden until the game ends.",
        ],
    },
    {
        type: "snakes_and_ladders",
        label: "Snakes & Ladders",
        tagline: "Pure dice luck. Climb the ladders, dodge the snakes, race to 100.",
        hubPath: "/games/snakes_and_ladders",
        newPath: "/games/snakes_and_ladders/new",
        detailPath: (id: string) => `/games/snakes_and_ladders/${id}`,
        available: true,
        howToPlay: [
            "Click Start a new snakes and ladders game, pick a player by username or from your mutual followers and send the invite. You move first.",
            "On your turn, press Roll. The server rolls a fair six-sided die and moves your token that many cells along the board.",
            "Land on the foot of a ladder and you climb to its top. Land on a snake's head and you slide down to its tail. It is all luck, no decisions.",
            "You must land exactly on 100 to win. If a roll would overshoot 100, you stay put and pass the turn.",
            "Games are correspondence-style with no clocks, so take as long as you like between rolls. Disconnects trigger a 60-second forfeit timer.",
            "Active games are public to spectators; finished games are archived under Past Games.",
        ],
    },
    {
        type: "pong",
        label: "Pong",
        tagline: "Real-time paddle duel. Aim with the paddle, not just block with it.",
        hubPath: "/games/pong",
        newPath: "/games/pong/new",
        detailPath: (id: string) => `/games/pong/${id}`,
        available: true,
        howToPlay: [
            "Click Start a new pong game, pick a player by username or from your mutual followers and send the invite. Once they accept you get a three second countdown, and then the ball is live.",
            "This one is real-time, not correspondence. Both of you are playing at once and the ball keeps moving whether you are watching or not, so stay at the court until the match ends.",
            "Steer your paddle with the mouse, with a finger on a touchscreen, or by holding the arrow keys. All three move the paddle at the same top speed, so nobody wins on their choice of device.",
            "Where on the paddle you hit the ball decides which way it leaves. Hit it near the centre and it comes back flat; hit it near an edge and it leaves at a steep angle. The paddle is an aiming device, not a wall, so pick your return rather than just reaching it.",
            "Every return speeds the ball up a little, until it reaches its top speed. A long rally ends up far faster than it started, which is what turns an edge-aimed shot into a winner.",
            "First to 7 points wins, but you must be 2 clear, so 7-6 keeps playing. If it is still level at 10-10, the next point takes it.",
            "If either player disconnects mid-match, they have 60 seconds to come back before they forfeit. The ball keeps moving in the meantime, so the connected player carries on scoring.",
            "Active games are public to spectators, who watch the same live court and have their own side chat that players cannot see. Finished games are archived under Past Games.",
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
