import { Button } from "../Button/Button";
import styles from "./DrawOfferBanner.module.css";

interface DrawOfferBannerProps {
    offeredByViewer: boolean;
    submitting: boolean;
    onAccept: () => void;
    onDecline: () => void;
}

export function DrawOfferBanner({ offeredByViewer, submitting, onAccept, onDecline }: DrawOfferBannerProps) {
    if (offeredByViewer) {
        return (
            <div className={styles.drawPending}>
                Draw offered. Waiting for your opponent to respond. It is withdrawn if you make a move.
            </div>
        );
    }

    return (
        <div className={styles.drawBanner}>
            <span className={styles.drawBannerText}>
                Your opponent has offered a draw. Accept to end the game as a draw, or decline to keep playing.
            </span>
            <div className={styles.drawBannerActions}>
                <Button variant="primary" size="small" onClick={onAccept} disabled={submitting}>
                    Accept draw
                </Button>
                <Button variant="ghost" size="small" onClick={onDecline} disabled={submitting}>
                    Decline
                </Button>
            </div>
        </div>
    );
}
