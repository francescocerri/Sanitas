// Package api contiene il contratto pubblico del servizio (OpenAPI),
// separato dal codice degli handler per poterlo versionare e
// referenziare (es. da CI o da altri servizi) senza dipendere da internal/.
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
