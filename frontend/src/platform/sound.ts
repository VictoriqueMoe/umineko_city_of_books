const MESSAGE_SOUND = "/sounds/message.wav";
const NOTIFICATION_SOUND = "/sounds/notification.wav";
const VOICE_JOIN_SOUND = "/sounds/voice-join.wav";
const VOICE_LEAVE_SOUND = "/sounds/voice-leave.wav";
const NOTIFICATION_SUPPRESS_MS = 1500;
const DEFAULT_VOLUME = 0.15;
const MAX_CACHED_AUDIO = 16;
const PINNED_SOUNDS = [MESSAGE_SOUND, NOTIFICATION_SOUND, VOICE_JOIN_SOUND, VOICE_LEAVE_SOUND];

const cache = new Map<string, HTMLAudioElement>();
const priming = new Set<HTMLAudioElement>();
let lastMessageSoundAt = 0;
let unlocked = false;

function evictOverflow(): void {
    for (const src of cache.keys()) {
        if (cache.size <= MAX_CACHED_AUDIO) {
            return;
        }
        if (PINNED_SOUNDS.includes(src)) {
            continue;
        }
        cache.delete(src);
    }
}

function ensureAudio(src: string): HTMLAudioElement {
    let audio = cache.get(src);
    if (!audio) {
        audio = new Audio(src);
        audio.preload = "auto";
        audio.load();
        cache.set(src, audio);
        evictOverflow();
    }
    return audio;
}

function unlockAudio(): void {
    if (unlocked) {
        return;
    }
    unlocked = true;
    const audios = Array.from(cache.values());
    for (let i = 0; i < audios.length; i++) {
        const audio = audios[i];
        const savedVolume = audio.volume;
        priming.add(audio);
        audio.muted = true;
        audio
            .play()
            .then(() => {
                if (!priming.delete(audio)) {
                    return;
                }
                audio.pause();
                audio.currentTime = 0;
                audio.muted = false;
                audio.volume = savedVolume;
            })
            .catch(() => {
                if (!priming.delete(audio)) {
                    return;
                }
                audio.muted = false;
                audio.volume = savedVolume;
            });
    }
}

const UNLOCK_EVENTS = ["click", "keydown", "touchstart"] as const;
for (let i = 0; i < UNLOCK_EVENTS.length; i++) {
    document.addEventListener(UNLOCK_EVENTS[i], unlockAudio, { once: true, passive: true });
}

export function playAudio(src: string, volume = DEFAULT_VOLUME): void {
    const audio = ensureAudio(src);
    priming.delete(audio);
    audio.muted = false;
    audio.volume = volume;
    if (audio.readyState > 0) {
        audio.currentTime = 0;
    }
    audio.play().catch(() => {});
}

ensureAudio(MESSAGE_SOUND);
ensureAudio(NOTIFICATION_SOUND);
ensureAudio(VOICE_JOIN_SOUND);
ensureAudio(VOICE_LEAVE_SOUND);

export function playMessageSound(): void {
    lastMessageSoundAt = Date.now();
    playAudio(MESSAGE_SOUND);
}

export function playNotificationSound(): void {
    if (Date.now() - lastMessageSoundAt < NOTIFICATION_SUPPRESS_MS) {
        return;
    }
    playAudio(NOTIFICATION_SOUND);
}

export function playVoiceJoinSound(): void {
    playAudio(VOICE_JOIN_SOUND, 0.3);
}

export function playVoiceLeaveSound(): void {
    playAudio(VOICE_LEAVE_SOUND, 0.3);
}
