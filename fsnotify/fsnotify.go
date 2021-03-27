package fsnotify

import (
	"context"
	"github.com/lifei6671/fsnotify-sync/internal/log"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
)

func Run(ctx context.Context) error {
	app := &cli.App{
		Name:  "fsnotify",
		Usage: "A file change monitoring synchronization tool.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "configuration file path",
				Value: "",
			},
		},
		Action: func(c *cli.Context) error {
			config := c.String("config")
			configFile, err := filepath.Abs(config)
			if err != nil {
				log.Logger.Errorf("解析配置文件路径失败 -> %s - %+v", config, err)
				return err
			}
			rules, err := LoadFromFile(configFile)
			if err != nil {
				log.Logger.Errorf("解析配置文件失败 -> %s - %+v", config, err)
				return err
			}
			//log.Logger.Fatal(rules)

			for _, rule := range rules {
				go func(rule *Rule) {
					err := rule.Watcher(ctx)
					if err != nil {
						log.Logger.Fatalf("监听目录失败 -> %s - %+v", rule.Root, err)
					}
				}(rule)
			}
			<-ctx.Done()
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		return err
	}
	return nil
}
