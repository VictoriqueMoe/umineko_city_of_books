import { Input } from "../../../../components/Input/Input";
import { Select } from "../../../../components/Select/Select";
import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { isEnabled } from "../../../../domain/siteSettings";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function StreamingSection({ form }: SectionProps) {
    const { settings, updateField, toggleField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Watch Parties, Voice &amp; Streaming</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Watch party: shared browser (Hyperbeam)</span>
                    <Input
                        type="password"
                        value={settings.hyperbeam_api_key ?? ""}
                        onChange={e => updateField("hyperbeam_api_key", e.target.value)}
                        fullWidth
                        placeholder="sk_test_..."
                    />
                    <span className={styles.fieldHint}>
                        Lets members watch a shared virtual browser together. Leave it empty to offer screen sharing
                        only, which uses the LiveKit credentials below.
                    </span>
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Shared browser region</span>
                    <Select
                        value={settings.hyperbeam_region ?? "EU"}
                        onChange={e => updateField("hyperbeam_region", e.target.value)}
                    >
                        <option value="NA">North America</option>
                        <option value="EU">Europe</option>
                        <option value="AS">Asia</option>
                    </Select>
                    <span className={styles.fieldHint}>
                        Where the shared browser runs. Pick the one nearest most of your members.
                    </span>
                </div>
                <ToggleSwitch
                    label="Enable Voice Chat"
                    description="Allow voice calls in chat rooms and DMs (requires a self-hosted LiveKit server)"
                    enabled={isEnabled(settings.voice_enabled)}
                    onChange={v => toggleField("voice_enabled", v)}
                />
                <ToggleSwitch
                    label="Enable Live Streaming"
                    description="Let members broadcast from OBS (WHIP) to a public /live page anyone can watch (requires the LiveKit ingress service)"
                    enabled={isEnabled(settings.streaming_enabled)}
                    onChange={v => toggleField("streaming_enabled", v)}
                />
                {(isEnabled(settings.voice_enabled) || isEnabled(settings.streaming_enabled)) && (
                    <>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>LiveKit URL</span>
                            <Input
                                value={settings.livekit_url ?? ""}
                                onChange={e => updateField("livekit_url", e.target.value)}
                                fullWidth
                                placeholder="wss://livekit.example.com"
                            />
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>API Key</span>
                            <Input
                                value={settings.livekit_api_key ?? ""}
                                onChange={e => updateField("livekit_api_key", e.target.value)}
                                fullWidth
                                placeholder="APIxxxxxxxx"
                            />
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>API Secret</span>
                            <Input
                                type="password"
                                value={settings.livekit_api_secret ?? ""}
                                onChange={e => updateField("livekit_api_secret", e.target.value)}
                                fullWidth
                                placeholder="secret"
                            />
                        </div>
                    </>
                )}
                {isEnabled(settings.streaming_enabled) && (
                    <>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Max Concurrent Streams</span>
                            <Input
                                type="number"
                                value={settings.stream_max_concurrent ?? ""}
                                onChange={e => updateField("stream_max_concurrent", e.target.value)}
                                fullWidth
                                placeholder="3"
                            />
                        </div>
                        <ToggleSwitch
                            label="Enable Smooth (HLS) playback"
                            description="Record each live broadcaster to HLS so viewers can pick a buffered, freeze-resistant stream a few seconds behind live (requires the LiveKit egress service)"
                            enabled={isEnabled(settings.stream_hls_enabled)}
                            onChange={v => toggleField("stream_hls_enabled", v)}
                        />
                        {isEnabled(settings.stream_hls_enabled) && (
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>HLS Output Directory</span>
                                <Input
                                    value={settings.stream_hls_output_dir ?? ""}
                                    onChange={e => updateField("stream_hls_output_dir", e.target.value)}
                                    fullWidth
                                    placeholder="/app/data/hls"
                                />
                            </div>
                        )}
                    </>
                )}
            </div>
        </div>
    );
}
