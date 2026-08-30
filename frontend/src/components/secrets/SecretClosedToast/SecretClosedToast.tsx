import { Link } from "react-router";
import { useSecretAnnouncementSync } from "../../../hooks/useSecretAnnouncementSync";
import { Toast } from "../../Toast/Toast";

export function SecretClosedToast() {
    const { announcement, dismiss } = useSecretAnnouncementSync();

    if (!announcement) {
        return null;
    }

    const name = announcement.solver.display_name || announcement.solver.username;

    return (
        <Toast variant="arcane" duration={10000} onDismiss={dismiss}>
            <Link to={`/secrets/${announcement.secret_id}`} style={{ color: "inherit" }}>
                Uu~{" "}
                <strong>
                    <bdi>{name}</bdi>
                </strong>{" "}
                solved{" "}
                <em>
                    <bdi>{announcement.secret_title}</bdi>
                </em>{" "}
                before you could. Try again next time.
            </Link>
        </Toast>
    );
}
