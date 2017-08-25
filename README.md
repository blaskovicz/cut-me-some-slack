# Cut Me Some Slack!
> Anonymous Slack web portal. Read and write messages, help-desk style.

## Deploying

```
$ heroku create my-slack-app
$ heroku config:set SLACK_TOKEN=xoxp-... # slack token with correct scopes
$ heroku config:set HEROKU_APP_DOMAIN=my-slack-app.herokuapp.com # domain for websockets
$ heroku config:set SLACK_CHANNEL=api-testing # default channel to display
$ heroku buildpacks:set heroku/go
$ git push heroku master
```

## TODO

* multi-channel support
* anonymous user posting
* avatars / ui overhaul
* message update / delete visualization
* threads?
* ...
