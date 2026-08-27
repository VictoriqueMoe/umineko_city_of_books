import { NewGameInvitePage } from "./NewGameInvitePage";

export function NewPongGamePage() {
    return (
        <NewGameInvitePage
            gameName="Pong"
            gameType="pong"
            blurb="Pick an opponent to invite. The match starts with a three second countdown, then the ball is live and you both steer your own paddle in real time."
        />
    );
}
