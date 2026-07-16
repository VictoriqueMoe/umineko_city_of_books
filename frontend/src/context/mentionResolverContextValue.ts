import { createContext } from "react";

export interface MentionResolverContextValue {
    isKnown: (username: string) => boolean | undefined;
    request: (username: string) => void;
}

export const MentionResolverContext = createContext<MentionResolverContextValue | null>(null);
