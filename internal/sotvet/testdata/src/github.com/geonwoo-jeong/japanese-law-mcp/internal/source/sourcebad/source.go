package sourcebad

import (
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery/testfixture" // want `SOT-ARCH-007`
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon/testfixture"         // want `SOT-ARCH-007`
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon/testfixture"    // want `SOT-ARCH-007`
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/mcp/testfixture"                    // want `SOT-ARCH-007`
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/testfixture"           // want `SOT-ARCH-007`
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/stdio/testfixture"        // want `SOT-ARCH-007`
	_ "github.com/modelcontextprotocol/go-sdk/mcp"                                            // want `SOT-ARCH-007`
)
