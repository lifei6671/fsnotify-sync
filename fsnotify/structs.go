package fsnotify

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/fsnotify/fsnotify"
	"github.com/lifei6671/fsnotify-sync/internal/ioutils"
	"github.com/lifei6671/fsnotify-sync/internal/log"
	"gopkg.in/ini.v1"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func LoadFromFile(configFile string) ([]*Rule, error) {
	cfg, err := ini.ShadowLoad(configFile)
	//err = yaml.Unmarshal(b, &rule)
	if err != nil {
		log.Logger.Errorf("解析配置文件失败 -> %s - %+v", configFile, err)
		return nil, err
	}
	rules := make([]*Rule, 0)
	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection || strings.Contains(section.Name(), ".") {
			continue
		}
		rule := &Rule{
			Name:      section.Name(),
			Gid:       cfg.Section("").Key("gid").MustInt(),
			Uid:       cfg.Section("").Key("uid").MustInt(),
			Perm:      uint32(cfg.Section("").Key("perm").MustInt64()),
			Recursion: cfg.Section("").Key("recursion").MustBool(true),
		}
		err = cfg.Section(section.Name()).MapTo(rule)
		if err != nil {
			log.Logger.Errorf("解析配置文件失败 -> %s - %+v", section.Name(), err)
			return nil, err
		}
		cur := cfg.Section(section.Name() + ".files")
		rule.Files = cur.KeysHash()
		rule.Ignore = cfg.Section(section.Name() + ".ignore").Key("ignore").ValueWithShadows()
		rules = append(rules, rule)
	}
	return rules, err
}

type Rule struct {
	Name      string              `ini:"-"`
	Root      string              `ini:"root"`
	Recursion bool                `ini:"recursion"`
	Gid       int                 `ini:"gid"`
	Uid       int                 `ini:"uid"`
	Perm      uint32              `ini:"perm"`
	Files     map[string]string   `ini:"files,,delim"`
	Ignore    []string            `ini:"ignore,omitempty,allowshadow"`
	ch        chan fsnotify.Event `ini:"-"`
	isInit    bool                `ini:"-"`
	watcher   *fsnotify.Watcher   `ini:"-"`
	eventTime *sync.Map
}

func (r *Rule) Watcher(ctx context.Context) error {
	if f, err := os.Stat(r.Root); err != nil || os.IsNotExist(err) || !f.IsDir() {
		return errors.New("指定的路径不存在或不是目录 ->" + r.Root)
	}
	r.init()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Logger.Errorf("创建监听器失败 -> %s - %+v", r.Root, err)
		return err
	}
	r.watcher = watcher
	if r.Recursion {
		err = filepath.WalkDir(r.Root, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				if r.ignore(path) {
					return nil
				}
				err = watcher.Add(path)
				if err != nil {
					log.Logger.Errorf("添加监听目录失败 -> %s - %+v", r.Root, err)
					return err
				}
				log.Logger.Infof("添加监听目录 -> %s", path)
			}
			return nil
		})
		if err != nil {
			log.Logger.Errorf("遍历目录失败 ->%s - %+v", r.Root, err)
		}
	} else {
		err = watcher.Add(r.Root)
		if err != nil {
			log.Logger.Errorf("添加监听目录失败 -> %s - %+v", r.Root, err)
			return err
		}
		log.Logger.Infof("添加监听目录 -> %s", r.Root)
	}
	defer ioutils.Close(watcher)
	go r.handler(ctx)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if r.ignore(event.Name) {
				log.Logger.Infof("跳过忽略文件 -> %s", event.Name)
				continue
			}
			if ioutils.IsDir(event.Name) {
				log.Logger.Infof("增加监听目录 -> %s", event.Name)
				_ = watcher.Add(event.Name)
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove ||
				event.Op&fsnotify.Chmod == fsnotify.Chmod ||
				event.Op&fsnotify.Rename == fsnotify.Rename {
				mt := ioutils.GetFileModTime(event.Name)
				if mt == r.getFileModTime(event.Name) {
					//log.Logger.Infof("跳过修改时间不变的文件 -> %s",event.Name)
					continue
				}
				r.eventTime.Store(event.Name, mt)

				log.Logger.Infof("file changed -> %s %d", event.Name, mt)
				r.ch <- event
			}
		case err := <-watcher.Errors:
			log.Logger.Errorf("出现未处理异常 -> %+v", r.Name, err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *Rule) handler(ctx context.Context) {
	for {
		select {
		case event, ok := <-r.ch:
			if !ok {
				return
			}
			dstFile := r.parse(event.Name)
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				err := os.Remove(dstFile)
				if err != nil {
					log.Logger.Errorf("删除文件失败 -> %s - %s - %+v", event.Name, dstFile, err)
				}
				continue
			}
			time.Sleep(time.Millisecond * 100)
			_ = r.createDir(dstFile)
			//如果是个目录则要创建目录
			if ioutils.IsFile(event.Name) {
				_, err := ioutils.CopyFile(dstFile, event.Name, fs.FileMode(r.Perm))
				if err != nil {
					log.Logger.Errorf("复制文件失败 -> %s - %s - %+v", event.Name, dstFile, err)
					continue
				}
			} else {
				continue
			}
			err := os.Chown(dstFile, r.Uid, r.Gid)
			if err != nil {
				log.Logger.Errorf("修改文件所有者失败 ->  %s - %s - %+v", event.Name, dstFile, err)
				continue
			}
			log.Logger.Infof("文件或目录复制成功->  %s -> %s", event.Name, dstFile)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Rule) init() {
	if r.isInit {
		return
	}
	r.eventTime = &sync.Map{}
	defer func() {
		r.isInit = true
	}()
	files := make(map[string]string, len(r.Files))

	for local, remote := range r.Files {
		files[filepath.Join(r.Root, local)] = remote
	}
	r.Files = files
	ignores := make([]string, len(r.Ignore))
	for i, filename := range r.Ignore {
		ignores[i] = filepath.Join(r.Root, filename)
	}
	r.Ignore = ignores
	if r.ch == nil {
		r.ch = make(chan fsnotify.Event, 500)
	}
}

//解析出来最匹配的最长路径.
func (r *Rule) parse(src string) string {
	var dstFile string
	var n int

	for local, remote := range r.Files {
		if ioutils.IsFile(local) {
			if filepath.Base(local) == filepath.Base(src) {
				return remote
			}
			return filepath.Join(remote, filepath.Base(local))
		}
		local = strings.TrimSuffix(local, "*")
		if strings.HasPrefix(src, local) && len(local) > n {
			n = len(local)
			dstFile = filepath.Join(remote, strings.TrimPrefix(src, local))
		}
	}
	return dstFile
}

//判断是否是忽略文件.
func (r *Rule) ignore(filename string) bool {
	for _, f := range r.Ignore {
		f = strings.TrimSuffix(f, "*")
		if strings.HasPrefix(filename, f) {
			return true
		}
	}
	return false
}

func (r *Rule) createDir(filename string) error {
	var dstDir string
	if ioutils.IsDir(filename) {
		dstDir = filename
	} else {
		dstDir = filepath.Dir(filename)
	}
	if ioutils.FileExist(dstDir) {
		return nil
	}
	err := os.MkdirAll(dstDir, fs.FileMode(r.Perm))
	if err != nil {
		log.Logger.Errorf("创建目录失败 -> %s - %s - %+v", filename, dstDir, err)
		return err
	}
	err = os.Chown(dstDir, r.Uid, r.Gid)
	if err != nil {
		log.Logger.Errorf("修改文件所有者失败 ->  %s - %s - %+v", filename, dstDir, err)
		return err
	}
	return err
}

func (r *Rule) getFileModTime(filename string) int64 {
	if v, ok := r.eventTime.Load(filename); ok {
		return v.(int64)
	}
	return 0
}

func (r *Rule) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}
