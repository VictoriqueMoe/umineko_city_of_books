import { useEffect } from "react";
import type { ScrollToBottomOptions } from "./useMessageHistory";

interface ChatViewportOptions {
    scrollToBottom: (opts?: ScrollToBottomOptions) => void;
}

export function useChatViewport(options: ChatViewportOptions) {
    const { scrollToBottom } = options;

    useEffect(() => {
        const vv = window.visualViewport;
        if (!vv) {
            return;
        }

        const root = document.documentElement;

        function applyVars() {
            const v = window.visualViewport;
            if (!v) {
                return;
            }

            root.style.setProperty("--chat-vh", `${v.height}px`);
            const inset = Math.max(0, window.innerHeight - v.height - v.offsetTop);
            root.style.setProperty("--kb-inset", `${inset}px`);
        }

        function onResize() {
            applyVars();
            scrollToBottom();
        }

        applyVars();
        vv.addEventListener("resize", onResize);
        vv.addEventListener("scroll", applyVars);
        return () => {
            vv.removeEventListener("resize", onResize);
            vv.removeEventListener("scroll", applyVars);
            root.style.removeProperty("--chat-vh");
            root.style.removeProperty("--kb-inset");
        };
    }, [scrollToBottom]);
}
