import { useResignGame, useSubmitGameAction } from "../../hooks/mutations/gameRoom";
import { MinesweeperBoardView } from "../../components/games/minesweeper/MinesweeperBoardView";
import type { MinesweeperState, MinesweeperStats } from "../../types/api";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function MinesweeperBoard({ room, viewer, isSpectator }: GameBoardProps<MinesweeperState, MinesweeperStats>) {
    const submitAction = useSubmitGameAction(room.id);
    const resign = useResignGame();

    async function handleAction(payload: Record<string, unknown>) {
        await submitAction.mutateAsync(payload);
    }

    return (
        <MinesweeperBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onAction={handleAction}
            onResign={async () => {
                await resign.mutateAsync(room.id);
            }}
        />
    );
}

export function MinesweeperGamePage() {
    return (
        <GameRoomShell
            gameName="Minesweeper"
            inviteCopy={name => (
                <>
                    <bdi>{name}</bdi> has invited you to a minesweeper match. Accept to start; you will play
                    simultaneously and race to clear the board.
                </>
            )}
            Board={MinesweeperBoard}
        />
    );
}
