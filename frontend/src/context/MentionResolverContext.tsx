import type { PropsWithChildren } from "react";
import { useMentionResolver } from "../hooks/queries/mentions";
import { MentionResolverContext } from "./mentionResolverContextValue";

export function MentionResolverProvider({ children }: PropsWithChildren) {
    const value = useMentionResolver();

    return <MentionResolverContext.Provider value={value}>{children}</MentionResolverContext.Provider>;
}
