import { Link } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { PieceTrigger } from "../../features/easterEgg";
import styles from "./NotFoundPage.module.css";

export function NotFoundPage() {
    usePageTitle("Lost in the Fragment");

    return (
        <div className={styles.page}>
            <div className={styles.code}>404</div>
            <h1 className={styles.title}>
                This fragment was never written <PieceTrigger pieceId="piece_11" />
            </h1>
            <p className={styles.blurb}>
                The witch shrugs. The page you asked for does not exist in any kakera she can see. Perhaps a broken
                link, perhaps a story that was never told, perhaps a trick of the Endless. Return to the city and try
                again.
            </p>
            <Link to="/" className={styles.cta}>
                Back to the City of Books
            </Link>
        </div>
    );
}
