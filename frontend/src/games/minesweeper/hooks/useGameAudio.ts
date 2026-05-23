import { useCallback, useEffect, useRef, useState } from "react";
import { CharacterId } from "../types";
import { getGameOverAudio, getStartAudio } from "../audio";

type AudioEvent = "start" | "win" | "lose";

interface GameAudio {
    playing: boolean;
    play: (event: AudioEvent) => void;
    stop: () => void;
}

function resolveUrl(
    event: AudioEvent,
    myCharacter: CharacterId | "",
    opponentCharacter: CharacterId | "",
): string | null {
    if (event === "start") {
        return getStartAudio(myCharacter);
    }
    return getGameOverAudio(myCharacter, opponentCharacter, event === "win");
}

export function useGameAudio(myCharacter: CharacterId | "", opponentCharacter: CharacterId | ""): GameAudio {
    const audioRef = useRef<HTMLAudioElement | null>(null);
    const charsRef = useRef({ myCharacter, opponentCharacter });
    const [playing, setPlaying] = useState(false);

    useEffect(() => {
        charsRef.current = { myCharacter, opponentCharacter };
    }, [myCharacter, opponentCharacter]);

    const stop = useCallback(() => {
        if (audioRef.current) {
            audioRef.current.pause();
            audioRef.current.src = "";
            audioRef.current = null;
        }
        setPlaying(false);
    }, []);

    const play = useCallback(
        (event: AudioEvent) => {
            stop();
            const { myCharacter: my, opponentCharacter: op } = charsRef.current;
            const url = resolveUrl(event, my, op);
            if (!url) {
                return;
            }
            const audio = new Audio(url);
            audio.volume = 0.75;
            audioRef.current = audio;
            audio.addEventListener("ended", () => {
                setPlaying(false);
            });
            audio
                .play()
                .then(() => {
                    setPlaying(true);
                })
                .catch(() => {
                    setPlaying(false);
                });
        },
        [stop],
    );

    useEffect(() => {
        return () => {
            if (audioRef.current) {
                audioRef.current.pause();
                audioRef.current.src = "";
                audioRef.current = null;
            }
        };
    }, []);

    return { playing, play, stop };
}
