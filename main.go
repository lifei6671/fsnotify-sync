package main

import (
	"context"
	"github.com/lifei6671/fsnotify-sync/fsnotify"
)

func main() {
	fsnotify.Run(context.Background())
}
