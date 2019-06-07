# Test Flakes Bot design

## Workflow

TODO(incfly): just an illustrative note, will move to the README.md once implemented.

High level, following tasks will be executed as periodic task from another Cloud Scheduler job via a separate webhook endpoint, `/flakeschaser`.

- Issues matching following conditions (might needs adjustment based on fitering results)
  - Regex matching in issue descrption or title, `.*test.*flakes` or with certain testing labels, `label/flake-tests`.
  - Not closed yet
  - `time.Now - last_updates_timestamp` > threath_hold, 3 days.
- Update the issue with comments "@user-id testing flakes needs your attetion."
- Send email to the assignee? Maybe later?

## Code Changes

- `plugins/flakeschaser` as pkg.
- Read spanner table to figure out the working set.
- Process working item by sending comments as Github client, Github throttler
might be needed.
- Might better to update the Spanner tables right away to reduce the chance
another server instance duplicates `flakes chaser` comments. Alternatively, just invoke `/sync` by itself to simplfy the code logic?

## Questions

- What's size of the data to be processed?
- How are Cloud scheduler invoking the function? Cloud PubSub as glue or what?
Need IAM permission to see the istio-testing Cloud Scheduler job...
- Local development, add a `make serve` option to run locally? Haven't find any
Cloud specific dependency yet...