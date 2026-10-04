// Package docs embeds the OpenAPI specification so it can be served
// directly by the HTTP server without reading from disk at runtime.
package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPISpec []byte
