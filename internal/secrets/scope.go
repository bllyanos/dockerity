// Package secrets implements Dockerity's built-in secret store: envelope
// encryption with per-scope data keys wrapped by the master key.
package secrets

import "strconv"

// Scope names the owner of a data key and its secrets, e.g. "env:3" or "app:1".
type Scope string

// EnvScope is the scope of an environment's secrets.
func EnvScope(envID int64) Scope { return Scope("env:" + strconv.FormatInt(envID, 10)) }

// AppScope is the scope of app-level secrets such as the deploy key.
func AppScope(appID int64) Scope { return Scope("app:" + strconv.FormatInt(appID, 10)) }
