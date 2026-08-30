import { Component, type ErrorInfo, type ReactNode } from "react";
import styles from "./RootErrorBoundary.module.css";

interface RootErrorBoundaryProps {
    onError: (error: unknown, componentStack: string | null) => void;
    children: ReactNode;
}

interface RootErrorBoundaryState {
    failed: boolean;
    detail: string;
}

const INTACT: RootErrorBoundaryState = { failed: false, detail: "" };

function detailOf(error: unknown): string {
    try {
        if (error instanceof Error) {
            return error.message === "" ? error.name : `${error.name}: ${error.message}`;
        }

        if (typeof error === "string") {
            return error;
        }
    } catch {}

    return "";
}

export class RootErrorBoundary extends Component<RootErrorBoundaryProps, RootErrorBoundaryState> {
    state: RootErrorBoundaryState = INTACT;

    static getDerivedStateFromError(error: unknown): RootErrorBoundaryState {
        return { failed: true, detail: detailOf(error) };
    }

    componentDidCatch(error: Error, info: ErrorInfo): void {
        try {
            this.props.onError(error, info.componentStack ?? null);
        } catch {}
    }

    handleRetry = (): void => {
        this.setState(INTACT);
    };

    handleReload = (): void => {
        globalThis.location.reload();
    };

    render(): ReactNode {
        if (!this.state.failed) {
            return this.props.children;
        }

        return (
            <div className={styles.page}>
                <div className={styles.card}>
                    <h1 className={styles.title}>The game board has shattered</h1>
                    <p className={styles.blurb}>
                        Something went wrong while this page was being drawn, so the witch has shown you the wreckage
                        rather than an empty room. Try again first, which keeps everything you still have open. Reload
                        only if the page refuses to come back.
                    </p>
                    {this.state.detail !== "" && (
                        <p dir="auto" className={styles.detail}>
                            {this.state.detail}
                        </p>
                    )}
                    <div className={styles.actions}>
                        <button
                            type="button"
                            className={`${styles.action} ${styles.primary}`}
                            onClick={this.handleRetry}
                        >
                            Try again
                        </button>
                        <button type="button" className={styles.action} onClick={this.handleReload}>
                            Reload the page
                        </button>
                        <a href="/" className={styles.action}>
                            Back to the City of Books
                        </a>
                    </div>
                </div>
            </div>
        );
    }
}
