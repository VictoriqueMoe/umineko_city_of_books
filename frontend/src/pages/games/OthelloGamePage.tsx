import { useResignGame, useSubmitGameAction } from "../../api/mutations/gameRoom";
import { OthelloBoardView } from "../../components/games/othello/OthelloBoardView";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function OthelloBoard({ room, viewer, isSpectator }: GameBoardProps) {
    const submitAction = useSubmitGameAction(room.id);
    const resign = useResignGame();

    async function handleMove(move: { square: string }) {
        await submitAction.mutateAsync({ square: move.square });
    }

    return (
        <OthelloBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onMove={handleMove}
            onResign={async () => {
                await resign.mutateAsync(room.id);
            }}
        />
    );
}

export function OthelloGamePage() {
    return (
        <GameRoomShell
            gameName="Othello"
            inviteCopy={name => `${name} has invited you to an othello game. Accept to start - you will play as white.`}
            Board={OthelloBoard}
        />
    );
}
