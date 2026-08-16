import { useResignGame, useSubmitGameAction } from "../../api/mutations/gameRoom";
import { SnakesAndLaddersBoardView } from "../../components/games/snakesandladders/SnakesAndLaddersBoardView";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function SnakesAndLaddersBoard({ room, viewer, isSpectator }: GameBoardProps) {
    const submitAction = useSubmitGameAction(room.id);
    const resign = useResignGame();

    async function handleRoll() {
        await submitAction.mutateAsync({ type: "roll" });
    }

    return (
        <SnakesAndLaddersBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onRoll={handleRoll}
            onResign={async () => {
                await resign.mutateAsync(room.id);
            }}
        />
    );
}

export function SnakesAndLaddersGamePage() {
    return (
        <GameRoomShell
            gameName="Snakes &amp; Ladders"
            inviteCopy={name =>
                `${name} has invited you to a game of snakes and ladders. Accept to start - you both race to square 100.`
            }
            Board={SnakesAndLaddersBoard}
        />
    );
}
