import { useSiteInfo } from "../../hooks/useSiteInfo";
import { HuntIcon } from "./HuntIcon";

interface HuntsInProgressProps {
    profileUserId: string;
}

export function HuntsInProgress({ profileUserId }: HuntsInProgressProps) {
    const siteInfo = useSiteInfo();
    const hunts = siteInfo.listed_secrets ?? [];
    return (
        <>
            {hunts.map(h => (
                <HuntIcon key={h.id} profileUserId={profileUserId} secret={h} />
            ))}
        </>
    );
}
