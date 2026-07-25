package modelbad

import (
	_ "encoding/xml" // want `SOT-ARCH-007`
	_ "net/http"     // want `SOT-ARCH-007`

	_ "github.com/japanese-law-mcp/japanese-law-mcp/internal/mcp/testfixture"             // want `SOT-ARCH-007`
	_ "github.com/japanese-law-mcp/japanese-law-mcp/internal/source/testfixture"          // want `SOT-ARCH-007`
	_ "github.com/japanese-law-mcp/japanese-law-mcp/internal/transport/stdio/testfixture" // want `SOT-ARCH-007`
	_ "github.com/modelcontextprotocol/go-sdk/mcp"                                        // want `SOT-ARCH-007`
)
