# Vox Showtime Check

[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/cr2007/vox-showtime-check)

A simple Go tool to monitor a VOX Cinema showtimes webpage and send notifications via [ntfy.sh](https://ntfy.sh/) when showtimes become available.

# Features
- Checks a configured URL for showtime listings
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

## How It Works
- Loads the last known state from `state.json` (created automatically).
- Fetches the configured showtimes URL.
- If showtimes are found and the previous state was "not found", sends a notification.
- If no showtimes are found, sends a periodic notification every 2 hours (configurable).
- Updates `state.json` accordingly.

## Customization
- Adjust the notification interval by changing `NotFoundInterval` in `showtimes/showtimes.go`.
- Modify the notification message or ntfy.sh headers in the same file.

# Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./.github/CONTRIBUTING.md) for guidelines on how to get started.
