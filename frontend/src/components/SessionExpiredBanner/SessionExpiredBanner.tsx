import { useSessionLost } from "../../hooks/useSessionLost";
import { reloadPage } from "../../platform/pageReload";
import { Banner, BannerButton } from "../Banner/Banner";

export function SessionExpiredBanner() {
    const lost = useSessionLost();

    function handleReload() {
        reloadPage();
    }

    if (!lost) {
        return null;
    }

    return (
        <Banner colour="red" role="alert" actions={<BannerButton onClick={handleReload}>Reload now</BannerButton>}>
            You have been signed out. Copy anything you were writing, then reload the page and sign in again.
        </Banner>
    );
}
