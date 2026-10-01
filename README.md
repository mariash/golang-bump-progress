# golang-bump-progress

App that tracks golang bump progress accross bosh releases on github, TAS/TASW/IST, docker images and plugins as configured in [config.json](./config.json).

To avoid github rate limiting set `GITHUB_TOKEN` value.

```
cf push golang-bump-progres --no-start
cf set-env golang-bump-progress GITHUB_TOKEN <some-github-token>
cf start golang-bump-progress
```
