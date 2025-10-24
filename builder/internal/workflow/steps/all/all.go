package all

import (
	_ "github.com/tobiaskrok/topi/builder/internal/workflow/steps/core"
	// _ "github.com/tobiaskrok/topi/builder/internal/workflow/steps/docker"
	_ "github.com/tobiaskrok/topi/builder/internal/workflow/steps/git"
	_ "github.com/tobiaskrok/topi/builder/internal/workflow/steps/setup"

	_ "github.com/tobiaskrok/topi/builder/internal/workflow/steps/artefact"
	// _ "github.com/tobiaskrok/topi/builder/internal/workflow/steps/shell"
)
