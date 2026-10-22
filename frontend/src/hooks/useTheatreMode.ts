import { useEffect, useState } from "react";
import { isEditableTarget } from "../utils/editableTarget";

export function useTheatreMode() {
    const [theatre, setTheatre] = useState(false);

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.defaultPrevented || event.repeat || event.ctrlKey || event.metaKey || event.altKey) {
                return;
            }

            if (event.key === "Escape") {
                setTheatre(false);
                return;
            }
            if ((event.key === "t" || event.key === "T") && !isEditableTarget(event.target)) {
                setTheatre(prev => !prev);
            }
        };

        window.addEventListener("keydown", onKeyDown);
        return () => window.removeEventListener("keydown", onKeyDown);
    }, []);

    return { theatre, setTheatre };
}
