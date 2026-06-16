# Vox Showtime Check

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/cr2007/vox-showtime-check)

A simple Go tool to monitor a VOX Cinema showtimes webpage and send notifications via [ntfy.sh](https://ntfy.sh/) when showtimes become available.

> [!WARNING]
> This project calls an undocumented, internal API of `voxcinemas.com`, reverse-engineered by inspecting the network requests the site's own frontend makes in a browser. It was built **for educational purposes only**, to demonstrate how a client-side-rendered site's data can be inspected and consumed directly. The endpoint, its parameters, and the API key it uses are not publicly documented, are not provided or sanctioned by VOX Cinemas, and may change or break without notice.
>
> Use this at your own risk and in accordance with VOX Cinemas' [Terms of Use](https://uae.voxcinemas.com/terms-and-conditions). The author(s) of this project take no responsibility for any misuse of this code, for any violation of VOX Cinemas' terms of service, or for any consequences (including, but not limited to, access being blocked or legal action) arising from its use. Do not use this to scrape data at scale, resell it, or otherwise abuse VOX's infrastructure.

# Features
- Checks a movie's availability and today's showtimes directly via VOX's internal API (see [How It Works](#how-it-works)).
- Sends notifications to an ntfy.sh topic when showtimes appear or at intervals if not found.
- Persists state to avoid multiple notifications.
- Designed for scheduled/automated use (e.g., via GitHub Actions).

# Setup

### 1. Prerequisites
- Go 1.18 or newer (recommended)
- An [ntfy.sh](https://ntfy.sh/) topic for notifications

### 2. Clone the Repository
```sh
git clone https://github.com/<your-username>/vox-showtime-check.git
cd vox-showtime-check

# Makes a copy of the '.env' file
cp .env.example .env
```

### 3. Configure Environment Variables
Create a `.env` file or set the following environment variables:

- `SHOWTIMES_URL`: The URL of the showtimes page to monitor (e.g., `https://uae.voxcinemas.com/movies/<movie>`).
- `NTFY_TOPIC`: The ntfy.sh topic to send notifications to (e.g., `my-ntfy-topic`).

Example `.env`:
```
SHOWTIMES_URL=https://uae.voxcinemas.com/movies/<movie>
NTFY_TOPIC=my-ntfy-topic
```

### 4. Run Locally
```sh
go run main.go
```

Or build and run:
```sh
go build -o vox-showtime-check main.go
./vox-showtime-check
```

# GitHub Actions Automation
This project includes a GitHub Actions workflow (`.github/workflows/check-showtimes.yml`) to run the check automatically:
- **Scheduled**: Runs every hour (on the hour, UTC)
- **On Push**: Runs when changes are pushed to the `main` branch
- **Manual**: Can be triggered via the Actions tab

The workflow uses [actions/cache](https://github.com/actions/cache) to persist the `state.json` file between runs, ensuring notification state is maintained.

### Required GitHub Actions Variables
Set the following repository variables in your GitHub repo settings:
- `SHOWTIMES_URL`
- `NTFY_TOPIC`

<!-- ## How It Works
VOX's movie pages (e.g. `https://uae.voxcinemas.com/movies/<movie>`) are rendered client-side: the showtimes block is populated by JavaScript *after* the page loads, so a plain HTML fetch never contains it. Instead, this tool calls the same internal API the site's own frontend calls, in four steps:

1. **Discover the frontend's `x-api-key`**: instead of hardcoding it, the tool reads it straight out of VOX's own JavaScript, the same way a browser does, by scanning the page's referenced `_next/static/chunks/*.js` files for the `apiKey:"..."` literal embedded in one of them. This key isn't a secret (every visitor's browser receives the same value), but VOX changes the chunk's content hash, and potentially the key, on every deploy, so hardcoding it would silently go stale. The chunk that worked last time is cached in `state.json` and tried first; a full rescan only happens on a cache miss (e.g. after a VOX deploy).
2. **Get a guest auth token**: `GET https://<region>-apife.voxcinemas.com/groups/authToken` (no credentials needed; this is what any anonymous browser receives).
3. **Resolve the movie slug to its internal code**: `GET .../v1/vox2-0/groups/Movies/<slug>`, which returns whether the movie is `Bookable` and its `HoCode`.
4. **Fetch today's sessions**: `GET .../v1/vox2-0/groups/api/Sessions/<REGION>/<HoCode>/<YYYY-MM-DD>`, returning every cinema and session for that date.

The `<region>` (e.g. `uae`) and `<slug>` (e.g. `obsession`) are both parsed out of `SHOWTIMES_URL`, so no extra configuration is needed. -->

Each run:
- Loads the last known state from `state.json` (created automatically).
- Calls the API above to check whether any sessions exist today.
- If showtimes are found and the previous state was "not found", sends a notification.
- If no showtimes are found, sends a periodic notification every 2 hours (configurable).
- Updates `state.json` accordingly (including the cached API-key chunk path).

## Customization
- Adjust the notification interval by changing `NotFoundInterval` in `showtimes/showtimes.go`.
- Modify the notification message in `showtimes/showtimes.go`, or the ntfy.sh delivery logic in `showtimes/ntfy.go`.

# Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./.github/CONTRIBUTING.md) for guidelines on how to get started.
