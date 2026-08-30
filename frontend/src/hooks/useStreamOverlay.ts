import { useCallback, useEffect, useState } from "react";
import {
    OVERLAY_CONNECTOR_FILENAME,
    useOverlayConnectorFile,
    useResetOverlayToken,
    useTestOverlay,
} from "./mutations/overlay";
import { useOverlayConnection } from "./queries/overlay";
import { downloadBlob } from "../utils/download";
import { errorMessage } from "../utils/errorMessage";
import { nonFatal } from "../utils/nonFatal";
import type { OverlayConnection } from "../types/api";

const COPIED_NOTICE_MS = 2000;

const RESET_PROMPT =
    "Reset your overlay token? Your current connector file will stop working and you'll need to download and re-import the new one.";

export interface StreamOverlay {
    connection: OverlayConnection | null;
    loading: boolean;
    loadError: string;
    actionError: string;
    success: string;
    copied: boolean;
    downloading: boolean;
    resetting: boolean;
    testing: boolean;
    downloadConnector: () => void;
    resetToken: () => void;
    sendTestOverlay: () => void;
    copyToken: () => void;
}

export function useStreamOverlay(): StreamOverlay {
    const { connection, loading, error: loadError } = useOverlayConnection();

    const { mutateAsync: fetchConnector, isPending: downloading } = useOverlayConnectorFile();
    const { mutateAsync: resetOverlayToken, isPending: resetting } = useResetOverlayToken();
    const { mutateAsync: testOverlay, isPending: testing } = useTestOverlay();

    const [actionError, setActionError] = useState("");
    const [success, setSuccess] = useState("");
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        if (!copied) {
            return;
        }

        const timer = window.setTimeout(() => setCopied(false), COPIED_NOTICE_MS);

        return () => window.clearTimeout(timer);
    }, [copied]);

    const downloadConnector = useCallback((): void => {
        setActionError("");

        fetchConnector()
            .then(blob => {
                downloadBlob(blob, OVERLAY_CONNECTOR_FILENAME);
            })
            .catch((thrown: unknown) => {
                setActionError(errorMessage(thrown, "Could not download the connector."));
            });
    }, [fetchConnector]);

    const resetToken = useCallback((): void => {
        if (!window.confirm(RESET_PROMPT)) {
            return;
        }

        setActionError("");
        setSuccess("");

        resetOverlayToken()
            .then(() => {
                setCopied(false);
                setSuccess("Token reset. Download the new connector below.");
            })
            .catch((thrown: unknown) => {
                setActionError(errorMessage(thrown, "Could not reset your token."));
            });
    }, [resetOverlayToken]);

    const sendTestOverlay = useCallback((): void => {
        setActionError("");
        setSuccess("");

        testOverlay()
            .then(() => {
                setSuccess("Test overlay sent. Check your SAMMI overlay.");
            })
            .catch((thrown: unknown) => {
                setActionError(
                    errorMessage(thrown, "Could not send a test overlay. Make sure SAMMI is open and connected."),
                );
            });
    }, [testOverlay]);

    const token = connection?.token ?? "";

    const copyToken = useCallback((): void => {
        if (token === "") {
            return;
        }

        navigator.clipboard
            .writeText(token)
            .then(() => {
                setCopied(true);
            })
            .catch(nonFatal);
    }, [token]);

    return {
        connection,
        loading,
        loadError,
        actionError,
        success,
        copied,
        downloading,
        resetting,
        testing,
        downloadConnector,
        resetToken,
        sendTestOverlay,
        copyToken,
    };
}
