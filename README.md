# Cut Me Some Slack!
> Anonymous Slack web portal. Read and write messages, help-desk style.


## Why?

Slack is convenient and easy to use, yet people don't always want to log in!
Simply present them with a link to your website, and _presto_, they can be chatting
with you in seconds!

## Developing

First create a slack tocken for your domain. Optionally, add
it to your .env at the root of the project for local development.


To start the React development server:

```
$ yarn install
$ # Optionally, export REACT_APP_SLACK_CHANNEL=api-testing
$ # Optionally, export REACT_APP_BACKEND_URI=ws://otherhost:otherport/stream 
$ PORT=3001 yarn start
```


To start the Golang backend:

```
$ PORT=3000 go run cmd/cut-me-some-slack/main.go
```

Then visit http://localhost:3001 for the development, hot-reloading,
React frontend or http://localhost:3000 for the production build (once
`yarn build` has been run).


## Deploying to Heroku

```
$ heroku create my-slack-app
$ heroku config:set SLACK_TOKEN=xoxp-... # slack token with correct scopes
$ heroku config:set HEROKU_APP_DOMAIN=my-slack-app.herokuapp.com # domain for websockets
$ heroku config:set REACT_APP_SLACK_CHANNEL=api-testing # default channel to display
$ heroku buildpacks:set heroku/go
$ heroku buildpacks:add heroku/nodejs
$ git push heroku master
```

## TODO

* channel config
* multi-channel support
* avatars / ui overhaul
* message update / delete visualization
* reactions
* better edit pane
* threads?
* ...
