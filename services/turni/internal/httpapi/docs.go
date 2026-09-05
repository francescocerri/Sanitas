package httpapi

import (
	_ "embed"
	"net/http"

	"github.com/francescocerri/sanitas/services/turni/api"
)

// La pagina Swagger UI carica lo script/CSS da CDN (jsdelivr) invece di
// vendorizzare swagger-ui-dist nel repo: evita di aggiungere una
// dipendenza Go e qualche MB di asset statici solo per la UI di
// documentazione, a costo di richiedere accesso a internet quando si
// consulta /docs (l'API stessa resta pienamente self-hosted).
//
//go:embed swagger-ui.html
var swaggerUIPage []byte

func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(api.OpenAPISpec)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(swaggerUIPage)
}
