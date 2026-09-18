import { useSessionLost } from "../../hooks/useSessionLost";
import { Banner, BannerButton } from "../Banner/Banner";

export function SessionExpiredBanner() {
    const lost = useSessionLost();

    if (!lost) {
        return null;
    }

    return (
        <Banner
            colour="red"
            role="alert"
            actions={
                <BannerButton
                    onClick={() => {
                        window.location.reload();
                    }}
                >
                    Reload now
                </BannerButton>
            }
        >
            You have been signed out. Copy anything you were writing, then reload the page and sign in again.
        </Banner>
    );
}
