import { useCallback, useState } from "react";

export function useVote(initialScore: number, initialUserVote: number, voteFn: (value: number) => Promise<void>) {
    const [score, setScore] = useState(initialScore);
    const [userVote, setUserVote] = useState(initialUserVote);
    const [prevInitialScore, setPrevInitialScore] = useState(initialScore);
    const [prevInitialUserVote, setPrevInitialUserVote] = useState(initialUserVote);

    if (prevInitialScore !== initialScore) {
        setPrevInitialScore(initialScore);
        setScore(initialScore);
    }
    if (prevInitialUserVote !== initialUserVote) {
        setPrevInitialUserVote(initialUserVote);
        setUserVote(initialUserVote);
    }

    const vote = useCallback(
        async (value: number) => {
            const newValue = value === userVote ? 0 : value;
            const oldScore = score;
            const oldVote = userVote;

            setScore(oldScore - oldVote + newValue);
            setUserVote(newValue);

            try {
                await voteFn(newValue);
            } catch {
                setScore(oldScore);
                setUserVote(oldVote);
            }
        },
        [score, userVote, voteFn],
    );

    return { score, userVote, vote };
}
