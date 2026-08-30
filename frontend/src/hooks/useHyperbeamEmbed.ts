import { useEffect, useRef, useState, type RefObject } from "react";

import {
    destroyHyperbeamEmbed,
    mountHyperbeamEmbed,
    setHyperbeamInputEnabled,
    type HyperbeamHandle,
} from "../api/hyperbeam/embed";
import { reportClientError } from "../api/telemetry";
import { errorMessage } from "../utils/errorMessage";

export const HYPERBEAM_MOUNT_FAILED = "Failed to connect to virtual browser";

const CONTAINER_STYLE = "position:absolute;inset:0;width:100%;height:100%;";

export interface UseHyperbeamEmbedOptions {
    embedURL: string;
    isOpen: boolean;
    hasControl: boolean;
    onIdentify: (identifier: string) => void;
}

export interface HyperbeamEmbedState {
    wrapRef: RefObject<HTMLDivElement | null>;
    mountError: string | null;
}

export function useHyperbeamEmbed({
    embedURL,
    isOpen,
    hasControl,
    onIdentify,
}: UseHyperbeamEmbedOptions): HyperbeamEmbedState {
    const wrapRef = useRef<HTMLDivElement | null>(null);
    const handleRef = useRef<HyperbeamHandle | null>(null);
    const identifyRef = useRef(onIdentify);
    const hasControlRef = useRef(hasControl);
    const [mountError, setMountError] = useState<string | null>(null);

    useEffect(() => {
        identifyRef.current = onIdentify;
    }, [onIdentify]);

    useEffect(() => {
        hasControlRef.current = hasControl;
    }, [hasControl]);

    useEffect(() => {
        if (!isOpen || !embedURL) {
            return;
        }

        const wrap = wrapRef.current;
        if (!wrap) {
            return;
        }

        const container = document.createElement("div");
        container.style.cssText = CONTAINER_STYLE;
        wrap.appendChild(container);

        let cancelled = false;
        let activeHandle: HyperbeamHandle | null = null;

        setMountError(null);

        mountHyperbeamEmbed({ container, embedURL, inputEnabled: hasControlRef.current })
            .then(handle => {
                if (cancelled) {
                    destroyHyperbeamEmbed(handle);
                    return;
                }

                activeHandle = handle;
                handleRef.current = handle;

                if (handle.userId) {
                    identifyRef.current(handle.userId);
                }
            })
            .catch((thrown: unknown) => {
                reportClientError(thrown, { source: "caught" });

                if (cancelled) {
                    return;
                }

                setMountError(errorMessage(thrown, HYPERBEAM_MOUNT_FAILED));
            });

        return () => {
            cancelled = true;
            destroyHyperbeamEmbed(activeHandle);
            handleRef.current = null;

            if (container.parentElement) {
                container.parentElement.removeChild(container);
            }
        };
    }, [embedURL, isOpen]);

    useEffect(() => {
        setHyperbeamInputEnabled(handleRef.current, hasControl);
    }, [hasControl]);

    return { wrapRef, mountError };
}
