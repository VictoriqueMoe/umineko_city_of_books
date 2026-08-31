# Mobile app (Capacitor)

Everything specific to the packaged native build: local iteration against a dev server, over-the-air bundles, the API base URL baked in at build time, and native push.

The same React frontend is packaged as a native iOS/Android app via [Capacitor](https://capacitorjs.com/). The Capacitor project lives in `frontend/` (config in `frontend/capacitor.config.ts`, generated native project in `frontend/android/`).

```bash
cd frontend
npm run build:app   # builds the app bundle into frontend/dist-app/ using .env.app
npm run cap:sync    # build:app + copy assets into the native projects
npm run cap:android # build:app + sync android + open Android Studio
npm run cap:local   # sync android against a live dev server, for local iteration
npm run build:ota   # build:app + sign and encrypt it into ../static/app-bundles/ as an OTA bundle (needs the Capgo private key)
npm run assets      # regenerate launcher icons and splash screens
```

The web build (`npm run build`) and the app build (`npm run build:app`) are separate artefacts: the web build goes to `../static/` (served by the Go server, same-origin), the app build goes to `dist-app/` (bundled into the app). iOS must be built on macOS (`npx cap add ios` then Xcode); Android builds on any OS with Android Studio installed.

## Local app development

`npm run cap:local` re-syncs the Android project with the Capacitor `server.url` pointed at a live Vite dev server, so the installed app loads your working tree instead of the bundled `dist-app/`. It defaults to `http://10.0.2.2:5173`, the emulator's alias for the host machine. Pass a LAN IP for a physical device:

```bash
npm run cap:local -- 192.168.1.50
```

Both `npm run dev` (Vite on `:5173`) and the Go backend (`:4323`) need to be running, and the app itself is launched from the IDE rather than by the script.

## Over-the-air bundles

The app ships `@capgo/capacitor-updater` with `autoUpdate` turned off and end-to-end bundle signing turned on. `npm run build:ota` zips `dist-app/` with the Capgo CLI, encrypts it with the private signing key, and writes `../static/app-bundles/<VITE_APP_VERSION>.zip` plus a `latest.json` manifest carrying the RSA-encrypted checksum and the AES session key; the Go server serves both from the embedded static bundle like any other asset. The matching public key is committed in `frontend/capacitor.config.ts` and compiled into the APK by `cap sync`, so a device only accepts a bundle that was produced with the private key, whatever the origin or edge serves. On the client, `frontend/src/utils/appUpdate.ts` fetches `/app-bundles/latest.json` on launch and on resume, refuses a manifest without `checksum` and `session_key`, downloads a newer version in the background, and stages it for the next start rather than swapping under the user.

The private key is never in the repo. `make-ota` reads it from `CAPGO_PRIVATE_KEY_FILE` (default `/run/secrets/capgo_private_key`); `Dockerfile.ci` mounts it as a BuildKit secret that `deploy.yml` fills from the `CAPGO_PRIVATE_KEY` repository secret, and that mount is `required=true`, so a CI build without the key fails rather than shipping unsigned. The plain `Dockerfile` mounts the same secret optionally: without it `make-ota` logs a warning and writes no bundle at all, so a local image simply offers no OTA update. To sign locally, point `CAPGO_PRIVATE_KEY_FILE` at your copy of `.capgo_key_v2`. Rotating the key means a new APK for everyone, because devices on the old public key cannot verify bundles signed with the new one.

## Changing the app's API base URL

The website's base URL is dynamic (`base_url` site setting). The packaged **app** is different: it has no server origin, so it needs an absolute API URL that is baked into the bundle at build time.

That URL comes from `VITE_API_BASE` in `frontend/.env.app`, the mode file that `npm run build:app` (`vite build --mode app`) loads:

```
VITE_API_BASE=https://whentheycry.social
```

To point the app at a different domain:

1. Edit `VITE_API_BASE` in `frontend/.env.app`.
2. Rebuild the app: `npm run build:app` (or `cap:android`).
3. Ship a new app version to the stores.

`frontend/.env.app` is committed; `frontend/.env` is gitignored and is your personal local override. Vite loads `.env` in every mode and lets the mode file win, so editing `.env` will not change what the app bundle points at, but it will change what a plain `npm run build` bakes into the website. Keep `frontend/.env` to dev values, or leave `VITE_API_BASE` out of it entirely so the website falls back to same-origin relative URLs.

Because the value is compiled into the bundle, already-installed apps keep the old domain until users update. If you migrate domains, keep the old one reachable (even as a redirect/proxy to the API) until old installs age out. The backend allows the app to connect cross-origin via `config.IsAppOrigin` (the fixed Capacitor webview origins) in addition to `base_url`.

## Native push notifications

The packaged app registers an FCM device token per install and receives notifications natively, so the phone still buzzes when the app is closed and no WebSocket is open. The website is unaffected either way.

- Two things are required: the `push_enabled` site setting turned on in admin, **and** an `FCM_CREDENTIALS_FILE` env var pointing at a Firebase service-account JSON mounted read-only into the container. With the setting on and no credentials file, the app logs a warning at startup and native push simply stays off.
- The Firebase client is rebuilt whenever `push_enabled` changes, so switching it on does not need a restart.
- Tokens are registered and unregistered through the API and stored per user and per device, so each install is addressed on its own.
