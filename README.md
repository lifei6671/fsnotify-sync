# 文件变更监控复制工具

实现监听指定目录的文件，当文件变动时更加规则复制文件到指定位置

## 使用

```bash
fsnotify-sync --config=conf/app.ini
```

## 配置文件

```ini
#是否递归监听目录
recursion=true
#新目录或文件的gid
gid=500
#新目录或文件的uid
uid=500
#目录或文件的权限值
file_perm=0655
dir_perm=0755
#项目名称
[fsnotify-sync]
#项目根目录
root=/data/fsnotify-sync
[fsnotify-sync.ignore]
#需要忽略的文件或目录
ignore=.git/
ignore=.idea/
[fsnotify-sync.files]
#需要映射的文件或目录
conf=/home/root/app/fsnotify-sync
```

