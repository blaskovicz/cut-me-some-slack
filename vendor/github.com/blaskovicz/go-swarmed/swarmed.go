package swarmed

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const secretPath = "/run/secrets"

/* LoadSecrets translates all files in /var/secrets to a corresponding env var in the process.
   For example, /var/secrets/db_password (with contents password) would be translated to
	 env variable DB_PASSWORD with value password.

	 This is a feature most useful to [Docker swarm](https://docs.docker.com/engine/swarm/secrets/)
	 users.
*/
func LoadSecrets() error {
	f, err := os.Open(secretPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	files, err := f.Readdirnames(0)
	if err != nil {
		return err
	}

	for _, f := range files {
		f = path.Base(f)
		b, err := ioutil.ReadFile(path.Join(secretPath, f))
		if err != nil {
			return err
		}
		// TODO: support prefix and override warning?
		err = os.Setenv(strings.ToUpper(f), string(b))
		if err != nil {
			return err
		}
	}

	return nil
}
