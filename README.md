# git-ui
api and ui layer for git

## Run

GIT_UI_DIR="/home/rsimpson/github/repo/" go run .

## Example api calls

```
curl localhost:8111/api/state | jq
curl localhost:8111/api/status | jq
curl localhost:8111/api/log | jq
curl localhost:8111/api/state | jq
curl localhost:8111/api/log | jq
curl localhost:8111/api/log?limit=3 | jq
curl localhost:8111/api/log?limit=3 | jq
curl localhost:8111/api/branch | jq
curl localhost:8111/api/log?limit=3 | jq
curl localhost:8111/api/log?limit=3 | jq
curl -X POST -d '{"branch":"clean-run"}' localhost:8111/api/checkout
curl localhost:8111/api/state | jq
curl localhost:8111/api/log?limit=3&branch=main | jq
curl localhost:8111/api/log?limit=3 | jq
curl localhost:8111/api/log?limit=3&branch=clean-run | jq
curl localhost:8111/api/log?branch=clean-run | jq
curl localhost:8111/api/log?branch=main&limit=3 | jq
curl 'localhost:8111/api/log?branch=main&limit=3' | jq
```

## Web url
http://localhost:8111/html/
