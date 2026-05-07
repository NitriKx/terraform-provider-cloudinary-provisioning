//go:build tools

package tools

import (
	// tfplugindocs generates provider documentation from schema and examples.
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
