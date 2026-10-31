# Vox Showtime Check

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/cr2007/vox-showtime-check)

A simple Go tool to monitor a VOX Cinema showtimes webpage and send notifications via [ntfy.sh](https://ntfy.sh/) when showtimes become available.

> [!WARNING]
> This project reads the public movie pages of `voxcinemas.com` and checks them for bookable sessions. It was built **for educational purposes only**. It is not provided or sanctioned by VOX Cinemas, and it can break without notice when VOX changes its page layout.
>
> Use this at your own risk and in accordance with VOX Cinemas' [Terms of Use](https://uae.voxcinemas.com/terms-and-conditions). The author(s) of this project take no responsibility for any misuse of this code, for any violation of VOX Cinemas' terms of service, or for any consequences (including, but not limited to, access being blocked or legal action) arising from its use. Do not use this to scrape data at scale, resell it, or otherwise abuse VOX's infrastructure.

# Features
- Checks a movie's VOX page for bookable sessions (see [How It Works](#how-it-works)).
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
- **Scheduled**: Runs four times each hour, at minutes 7, 22, 37 and 52 (UTC)
- **On Push**: Runs when changes are pushed to the `master` branch
- **Manual**: Can be triggered via the Actions tab

The workflow uses [actions/cache](https://github.com/actions/cache) to persist the `state.json` file between runs, ensuring notification state is maintained. If a run fails, it sends one "Showtime check failed" alert to your ntfy.sh topic.

See [docs/github-actions.md](./docs/github-actions.md) for the schedule, the state cache, and the failure alerts.

### Required GitHub Actions Variables
Set the following repository variables in your GitHub repo settings:
- `SHOWTIMES_URL`
- `NTFY_TOPIC`

## How It Works
VOX's movie pages (e.g. `https://uae.voxcinemas.com/movies/<movie>`) are rendered on the server. When booking is open, the page lists every session as a link of the form `/booking/<cinema>-<session>`. Before booking opens, the page has no such links. The tool fetches the page and looks for at least one session link.

If the page returns an error status, or VOX moves back to a client-rendered page (where sessions are not in the HTML), the run reports an error instead of "no showtimes".

Each run:
- Loads the last known state from `state.json` (created automatically).
- Fetches the movie page and checks it for session links.
- If showtimes are found and the previous state was "not found", sends a notification.
- If no showtimes are found, sends a periodic notification every 2 hours (configurable).
- Updates `state.json` accordingly.

## Customization
- Adjust the notification interval by changing `NotFoundInterval` in `showtimes/showtimes.go`.
- Modify the notification message in `showtimes/showtimes.go`, the ntfy.sh delivery logic in `showtimes/ntfy.go`, or the page check in `showtimes/page.go`.

# Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./.github/CONTRIBUTING.md) for guidelines on how to get started.
