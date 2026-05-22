package static

import "embed"

//go:embed openapi.yaml docs.html
var FS embed.FS
