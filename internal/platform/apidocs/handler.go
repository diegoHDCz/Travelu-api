// Package apidocs serves the embedded OpenAPI specification and a Swagger UI
// page, so the API is self-documenting without any external tooling.
package apidocs

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/docs"
)

// RegisterRoutes wires the public documentation endpoints onto mux: the raw
// OpenAPI spec and a Swagger UI page that renders it. Both are unauthenticated,
// mirroring /health.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /docs/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(docs.OpenAPISpec)
	})

	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerUIPage))
	})
}

const swaggerUIPage = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8" />
  <title>travelu-api &middot; docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/docs/openapi.yaml",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis],
    });
  </script>
</body>
</html>
`
