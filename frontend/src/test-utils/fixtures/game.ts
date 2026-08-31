import type { GameRoom, GameRoomPlayer, MinesweeperState, SpectatorMessage } from "../../types/api";

export function makeGamePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "player-0";

    return {
        user_id: id,
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        role: "player",
        slot: 0,
        joined: true,
        connected: true,
        user: { id, username: "battler", display_name: "Battler" },
        ...overrides,
    };
}

export function makeGameRoom<TState, TStats = unknown>(
    state: TState,
    overrides: Partial<GameRoom<TState, TStats>> = {},
): GameRoom<TState, TStats> {
    return {
        id: "room-1",
        game_type: "chess",
        status: "active",
        state,
        created_by: "player-0",
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        players: [],
        watcher_count: 0,
        ...overrides,
    };
}

export function makeSpectatorMessage(overrides: Partial<SpectatorMessage> = {}): SpectatorMessage {
    return {
        id: "m1",
        user_id: "u-two",
        user: { id: "u-two", username: "beatrice", display_name: "Beatrice" },
        body: "the golden truth",
        created_at: "2026-08-02T10:00:00.000Z",
        ...overrides,
    };
}

export function makeMinesweeperState(overrides: Partial<MinesweeperState> = {}): MinesweeperState {
    return {
        phase: "playing",
        width: 10,
        height: 10,
        mine_count: 10,
        characters: ["bernkastel", "erika"],
        revealed: [[], []],
        flagged: [[], []],
        revealed_count: [0, 0],
        values: [[], []],
        mines_placed: true,
        pending_clicks: [null, null],
        ...overrides,
    };
}
