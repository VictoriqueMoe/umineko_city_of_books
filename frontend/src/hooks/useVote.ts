import { useCallback, useRef } from "react";
import { useSyncedState } from "./useResetOnChange";

export function useVote(initialScore: number, initialUserVote: number, voteFn: (value: number) => Promise<void>) {
    const [score, setScore] = useSyncedState(initialScore);
    const [userVote, setUserVote] = useSyncedState(initialUserVote);
    const latestRequest = useRef(0);

    const vote = useCallback(
        async (value: number) => {
            const newValue = value === userVote ? 0 : value;
            const oldScore = score;
            const oldVote = userVote;
            const requestId = latestRequest.current + 1;
            latestRequest.current = requestId;

            setScore(oldScore - oldVote + newValue);
            setUserVote(newValue);

            try {
                await voteFn(newValue);
            } catch {
                if (latestRequest.current !== requestId) {
                    return;
                }

                setScore(oldScore);
                setUserVote(oldVote);
            }
        },
        [score, userVote, voteFn, setScore, setUserVote],
    );

    return { score, userVote, vote };
}
