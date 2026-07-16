import { useContext, useEffect } from "react";
import { Link } from "react-router";
import { MentionResolverContext } from "../../context/mentionResolverContextValue";

interface MentionLinkProps {
    username: string;
    label: string;
}

export function MentionLink({ username, label }: MentionLinkProps) {
    const resolver = useContext(MentionResolverContext);

    useEffect(() => {
        resolver?.request(username);
    }, [resolver, username]);

    if (resolver?.isKnown(username) !== true) {
        return <>{label}</>;
    }

    return <Link to={`/user/${username}`}>{label}</Link>;
}
