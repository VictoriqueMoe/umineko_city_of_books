import { useAuth } from "../../hooks/useAuth";
import { Banner, BannerLink } from "../Banner/Banner";

export function LockBanner() {
    const { user } = useAuth();
    if (!user || !user.locked) {
        return null;
    }

    return (
        <Banner colour="amber" actions={<BannerLink to="/users">Find a moderator</BannerLink>}>
            Your account is locked. You can still read the site and send direct messages to site staff.
            {user.lock_reason ? ` Reason: ${user.lock_reason}` : ""}
        </Banner>
    );
}
