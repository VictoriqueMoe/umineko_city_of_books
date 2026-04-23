const MESSAGE_SOUND = "/sounds/message.wav";
const NOTIFICATION_SOUND = "/sounds/notification.wav";
const NOTIFICATION_SUPPRESS_MS = 1500;
const DEFAULT_VOLUME = 0.15;

const cache = new Map<string, HTMLAudioElement>();
let lastMessageSoundAt = 0;
let unlocked = false;

function ensureAudio(src: string): HTMLAudioElement {
    let audio = cache.get(src);
    if (!audio) {
        audio = new Audio(src);
        audio.preload = "auto";
        audio.load();
        cache.set(src, audio);
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
        audio.muted = true;
        audio
            .play()
            .then(() => {
                audio.pause();
                audio.currentTime = 0;
                audio.muted = false;
                audio.volume = savedVolume;
            })
            .catch(() => {
                audio.muted = false;
                audio.volume = savedVolume;
            });
    }
}

const UNLOCK_EVENTS = ["click", "keydown", "touchstart"] as const;
for (let i = 0; i < UNLOCK_EVENTS.length; i++) {
    document.addEventListener(UNLOCK_EVENTS[i], unlockAudio, { once: true, passive: true });
}

function play(src: string, volume = DEFAULT_VOLUME): void {
    const audio = ensureAudio(src);
    audio.volume = volume;
    if (audio.readyState > 0) {
        audio.currentTime = 0;
    }
    audio.play().catch(() => {});
}

ensureAudio(MESSAGE_SOUND);
ensureAudio(NOTIFICATION_SOUND);

export function playMessageSound(): void {
    lastMessageSoundAt = Date.now();
    play(MESSAGE_SOUND);
}

export function playNotificationSound(): void {
    if (Date.now() - lastMessageSoundAt < NOTIFICATION_SUPPRESS_MS) {
        return;
    }
    play(NOTIFICATION_SOUND);
}

export function playRemoteAudio(url: string, volume = DEFAULT_VOLUME): void {
    play(url, volume);
}
