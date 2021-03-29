package main

import (
	"context"
	"github.com/lifei6671/fsnotify-sync/fsnotify"
	"github.com/lifei6671/fsnotify-sync/internal/log"
)

func main() {
	err := fsnotify.Run(context.Background())
	if err != nil {
		log.Logger.Fatalf("启动文件监听失败 -> %+v", err)
	}
}
