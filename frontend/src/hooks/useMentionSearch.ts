import { useCallback } from "react";
import { reportClientError } from "../api/telemetry";
import { rankMentionCandidates } from "../domain/mentionRanking";
import type { User } from "../types/api";
import { fetchSearchUsers } from "./queries/user";

export interface MentionSuggestion extends User {
    viewer_follows: boolean;
    follows_viewer: boolean;
}

export type MentionSearch = (query: string) => Promise<MentionSuggestion[]>;

export function useMentionSearch(): MentionSearch {
    return useCallback<MentionSearch>(async query => {
        try {
            const users = (await fetchSearchUsers(query)) as MentionSuggestion[];

            return rankMentionCandidates(users);
        } catch (thrown) {
            reportClientError(thrown, { source: "recoverable" });

            return [];
        }
    }, []);
}
