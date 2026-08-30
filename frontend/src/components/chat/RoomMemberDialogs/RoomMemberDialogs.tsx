import type { ChatRoomMember } from "../../../types/api";
import { Button } from "../../Button/Button";
import styles from "./RoomMemberDialogs.module.css";

export type RoomMemberDialogsVariant = "desktop" | "mobile";

export interface NicknameDialogProps {
    target: ChatRoomMember | null;
    value: string;
    error: string;
    saving: boolean;
    onValueChange: (value: string) => void;
    onCancel: () => void;
    onSave: () => void;
}

export interface TimeoutDialogProps {
    target: ChatRoomMember | null;
    amount: string;
    unit: string;
    error: string;
    saving: boolean;
    onAmountChange: (value: string) => void;
    onUnitChange: (value: string) => void;
    onCancel: () => void;
    onSave: () => void;
}

interface RoomMemberDialogsProps {
    variant: RoomMemberDialogsVariant;
    nickname: NicknameDialogProps;
    timeout: TimeoutDialogProps;
}

const NICKNAME_MAX = 32;

const TIMEOUT_UNITS = ["seconds", "hours", "weeks", "years", "decades", "centuries"];

export function RoomMemberDialogs({ variant, nickname, timeout }: RoomMemberDialogsProps) {
    const mobile = variant === "mobile";
    const overlayClass = `${styles.overlay} ${mobile ? styles.overlayMobile : styles.overlayDesktop}`;
    const dialogClass = `${styles.dialog} ${mobile ? styles.dialogMobile : styles.dialogDesktop}`;
    const rowClass = mobile ? styles.rowMobile : styles.rowDesktop;
    const errorClass = mobile ? styles.errorMobile : styles.errorDesktop;

    return (
        <>
            {nickname.target && (
                <div className={overlayClass} onClick={nickname.onCancel}>
                    <div className={dialogClass} onClick={e => e.stopPropagation()}>
                        <h3>
                            Change nickname for <bdi>{nickname.target.user.display_name}</bdi>
                        </h3>
                        <input
                            type="text"
                            dir="auto"
                            value={nickname.value}
                            maxLength={NICKNAME_MAX}
                            onChange={e => nickname.onValueChange(e.target.value)}
                            placeholder="Nickname (leave blank to clear)"
                            autoFocus
                        />
                        {nickname.error && <div className={errorClass}>{nickname.error}</div>}
                        <div className={styles.actions}>
                            <Button variant="ghost" size="small" onClick={nickname.onCancel} disabled={nickname.saving}>
                                Cancel
                            </Button>
                            <Button variant="primary" size="small" onClick={nickname.onSave} disabled={nickname.saving}>
                                {nickname.saving ? "Saving..." : "Save"}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {timeout.target && (
                <div className={overlayClass} onClick={timeout.onCancel}>
                    <div className={dialogClass} onClick={e => e.stopPropagation()}>
                        <h3>
                            Set timeout for <bdi>{timeout.target.user.display_name}</bdi>
                        </h3>
                        <div className={rowClass}>
                            <input
                                type="number"
                                min={1}
                                step={1}
                                value={timeout.amount}
                                onChange={e => timeout.onAmountChange(e.target.value)}
                                autoFocus
                            />
                            <select value={timeout.unit} onChange={e => timeout.onUnitChange(e.target.value)}>
                                {TIMEOUT_UNITS.map(unit => (
                                    <option key={unit} value={unit}>
                                        {unit}
                                    </option>
                                ))}
                            </select>
                        </div>
                        {timeout.error && <div className={errorClass}>{timeout.error}</div>}
                        <div className={styles.actions}>
                            <Button variant="ghost" size="small" onClick={timeout.onCancel} disabled={timeout.saving}>
                                Cancel
                            </Button>
                            <Button variant="danger" size="small" onClick={timeout.onSave} disabled={timeout.saving}>
                                {timeout.saving ? "Saving..." : "Set timeout"}
                            </Button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
}
