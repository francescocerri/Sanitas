// Package schema embeds turni's own SQL schema, so it can be applied by the
// service's own startup (with retry, since anagrafica.users must already
// exist for the FK — see docs/adr/0019) instead of
// docker-entrypoint-initdb.d, which ran before either service's binary
// ever started and offered no such retry.
package schema

import _ "embed"

//go:embed schema.sql
var SQL string
