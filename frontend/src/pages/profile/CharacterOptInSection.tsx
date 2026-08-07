import { useState } from "react";
import { useChatbotOptIn } from "../../api/queries/auth";
import { useUpdateChatbotOptIn } from "../../api/mutations/auth";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import styles from "./SettingsPage.module.css";

export function CharacterOptInSection() {
    const siteInfo = useSiteInfo();
    const restricted = siteInfo.chatbot_require_permission;
    const available = restricted && siteInfo.chatbot_enabled;
    const { optedIn, loading } = useChatbotOptIn(restricted);
    const optInMutation = useUpdateChatbotOptIn();
    const [error, setError] = useState("");

    function handleChange(next: boolean) {
        setError("");
        optInMutation.mutate(next, {
            onError: (e: Error) => setError(e.message),
        });
    }

    if (!restricted) {
        return null;
    }

    return (
        <div className={`${styles.section} ${styles.gridFull}`}>
            <h3 className={styles.sectionTitle}>Characters</h3>
            {error && <div className={styles.error}>{error}</div>}
            <ToggleSwitch
                enabled={optedIn}
                onChange={handleChange}
                disabled={!available || loading || optInMutation.isPending}
                label="Talk To Characters"
                description="Let character accounts answer you when you mention or reply to them in chat and on the game board"
            />
            {available ? (
                <p className={styles.mutedText}>
                    Opting in gives you the role that carries this, so any colour or badge the role also carries comes
                    with it.
                </p>
            ) : (
                <p className={styles.mutedText}>Characters are switched off across the site at the moment.</p>
            )}
        </div>
    );
}
