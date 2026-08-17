import { useResignGame, useSubmitGameAction } from "../../api/mutations/gameRoom";
import { MinesweeperBoardView } from "../../components/games/minesweeper/MinesweeperBoardView";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function MinesweeperBoard({ room, viewer, isSpectator }: GameBoardProps) {
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
            inviteCopy={name =>
                `${name} has invited you to a minesweeper match. Accept to start; you will play simultaneously and race to clear the board.`
            }
            Board={MinesweeperBoard}
        />
    );
}
