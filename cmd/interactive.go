package cmd

import (
	"github.com/fijdemon/gssh/internal/ui"
)

// RunInteractive 运行交互式界面
func RunInteractive() error {
	return ui.Run()
}

