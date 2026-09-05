package httpapi

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	// Effetto collaterale: registra la spec generata da swag (api/docs.go,
	// derivata dalle annotazioni sopra gli handler) nel registro letto da
	// http-swagger. Rigenerare con `swag init` dopo ogni modifica alle
	// annotazioni — c'è un controllo in CI che verifica che non ci si dimentichi.
	_ "github.com/francescocerri/sanitas/services/turni/api"
)

// docsHandler serve la Swagger UI (asset incorporati nella libreria,
// nessuna dipendenza da CDN esterno) su /docs/, con la spec generata.
func docsHandler() http.Handler {
	return httpSwagger.Handler(httpSwagger.URL("/docs/doc.json"))
}
