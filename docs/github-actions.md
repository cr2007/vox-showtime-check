# GitHub Actions Workflow

The workflow `.github/workflows/check-showtimes.yml` runs the check on GitHub and keeps its state between runs.

## Triggers

| Trigger | When |
| --- | --- |
| Schedule | At minutes 7, 22, 37 and 52 of each hour (UTC) |
| Push | When you push to `master` |
| Manual | From the Actions tab, or with `gh workflow run "Check Showtimes"` |

### Why the schedule is not "every hour on the hour"

GitHub starts scheduled runs late when it has a high load, and it skips some runs. The load is highest at the start of each hour. Before this change, the hourly schedule at minute 0 had these results:

- A run started a median of 26 min after the hour.
- About 46% of the hours had no run.
- The longest gap between two runs was 6.5 h.

Four runs each hour, away from minute 0, make the gaps smaller. More runs do not cause more notifications, because the saved state controls the notifications (see [State](#state)).

GitHub disables scheduled workflows in a public repository after 60 days without activity in the repository. GitHub sends an email before it does this. To start the schedule again, enable the workflow in the Actions tab.

## Steps

1. **Restore cached state.** This step loads the most recent `state.json`.
2. **Run script.** This step runs `go run main.go`. The step fails, and the run fails, if the page check or the notification fails.
3. **Save updated JSON state.** This step saves `state.json` with the key `showtimes-state-<run id>`. If the script fails, this step does not run, so the old state stays.
4. **Alert on failure.** This step runs only when an earlier step fails. See [Failure alerts](#failure-alerts).

## State

The workflow keeps `state.json` in the [Actions cache](https://github.com/actions/cache). Each run saves a new cache entry. The next run loads the most recent entry, because its restore key never exists and the prefix `showtimes-state-` matches all entries.

The state controls the notifications:

- **Showtimes available.** This notification is sent one time, when the status changes from "not found" to "found".
- **No showtimes yet.** This notification is sent at most one time every 2 hours (`NotFoundInterval`).

GitHub removes a cache entry after 7 days without use. If the state is lost, the next run starts with a new state and can send one more notification.

To start again from a new state (for example, to get the "available" notification again), delete the state entries:

```sh
gh cache list --key showtimes-state- --json key -q '.[].key' | xargs -n1 gh cache delete
```

## Failure alerts

When a run fails, the workflow sends an ntfy.sh message to `NTFY_TOPIC`:

- **Title:** Showtime check failed
- **Priority:** 4
- **Click:** opens the log of the failed run

The workflow sends one alert for each break, not one for each failed run. The alert step sends a message only when the previous completed run passed. If the check continues to fail, you get no more alerts until a run passes and a new failure starts. GitHub also sends its usual "workflow failed" email for each failed run.

These problems cause a failure alert:

| Problem | Example error in the run log |
| --- | --- |
| The movie page does not exist, or the URL is wrong | `page returned status 404` |
| VOX blocks the GitHub runner | `page returned status 403` |
| VOX changes the page to client-rendered | `page is client-rendered, so its HTML has no sessions` |
| ntfy.sh does not accept the notification | `ntfy.sh returned status 500` |

To test the failure alert:

1. Set `SHOWTIMES_URL` to a movie that does not exist, for example `https://uae.voxcinemas.com/movies/no-such-movie`.
2. Start a run. Make sure that you get one "Showtime check failed" message.
3. Start a second run. Make sure that you get no message.
4. Set `SHOWTIMES_URL` to the correct URL and start a run. Make sure that the run passes.

## Repository variables

Set these in **Settings > Secrets and variables > Actions > Variables**:

| Variable | Example |
| --- | --- |
| `SHOWTIMES_URL` | `https://uae.voxcinemas.com/movies/<movie>` |
| `NTFY_TOPIC` | `my-ntfy-topic` |

The failure alert goes to the same `NTFY_TOPIC`.
