package cmd

import (
	"github.com/fijdemon/gssh/internal/sync"
)

// RunPush 执行推送操作
func RunPush() error {
	return sync.Push()
}

