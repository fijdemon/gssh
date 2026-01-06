package cmd

import (
	"github.com/fijdemon/gssh/internal/sync"
)

// RunPull 执行拉取操作
func RunPull() error {
	return sync.Pull()
}

