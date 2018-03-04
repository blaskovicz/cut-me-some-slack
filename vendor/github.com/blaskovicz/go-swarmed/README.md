# go-swarmed
> Load docker swarm secrets at runtime in golang.

# About

To load secrets at runtime, use the following code:

```
import "github.com/blaskovicz/go-swarmed"

func main() {
  err := swarmed.LoadSecrets()
  if err != nil {
    panic(err)
  }
}
```

LoadSecrets translates all files in /var/secrets to a corresponding env var in the process.
For example, /var/secrets/db_password (with contents password) would be translated to
env variable DB_PASSWORD with value password.

This is a feature most useful to [Docker swarm](https://docs.docker.com/engine/swarm/secrets/)
users.
