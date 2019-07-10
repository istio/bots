# Test Flakes Bot design

Test flakes hurts our development velocity. This `flakechaser` bots automatically nags
flaky test issues owners if

- Issue has a 'flaky' keyword in title or description.
- Hasn't been updated for 3 days.
- Created within last 180 days.

The bot scanning the Cloud Spanner database with matching criteria to find the
work set, and then update the issues via Github API.

Known Issue

if the bot is triggered again before the /sync is invoked, due to the
storage lag behind, the bot can nag flakey test issue again...
