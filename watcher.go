package main

import (
	"io/ioutil"
	"os"
	pathUtil "path"

	"github.com/go-fsnotify/fsnotify"
)

// Watcher watches for file system changes, and sends an update message on the
// Updates channel
type Watcher struct {
	fsWatcher *fsnotify.Watcher

	Updates    chan string
	cancelChan chan bool

	filterFunc func(string) bool
}

// NewWatcher creates a new (recursive, constant) Watcher with root in path.
// It will stop executing when it receives a message on the cancel channel.
func NewWatcher(path string, cancel chan bool, filterFunc func(string) bool) (watcher *Watcher, err error) {
	watcher = &Watcher{}
	watcher.fsWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher.Updates = make(chan string)
	watcher.cancelChan = cancel
	watcher.filterFunc = filterFunc
	watcher.watchDirectory(path)

	return watcher, nil
}

// Run starts executing the watcher
func (watcher *Watcher) Run() {
	for {
		select {
		case event := <-watcher.fsWatcher.Events:
			log.Debug(event)
			watcher.Updates <- event.Name
		case err := <-watcher.fsWatcher.Errors:
			log.Error(err)
		case <-watcher.cancelChan:
			if err := watcher.fsWatcher.Close(); err != nil {
				log.Error(err)
			}
			close(watcher.Updates)
			return
		}
	}
}

func isDirectory(path string) (bool, error) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return stat.IsDir(), nil
}

func (watcher *Watcher) watchDirectory(path string) {
	entries, err := ioutil.ReadDir(path)

	if os.IsNotExist(err) {
		return
	} else if err != nil {
		log.Errorf("ReadDir failed for %s: %s", path, err)
		return
	}

	if !watcher.filterFunc(path) {
		log.Debugf("Path '%s' was filtered", path)
		return
	}

	if err = watcher.fsWatcher.Add(path); err != nil {
		log.Errorf("Failed to watch %s: %s", path, err)
	} else {
		log.Debug("Watching", path)

		for _, entry := range entries {
			subDir := pathUtil.Join(path, entry.Name())
			if isDir, err := isDirectory(subDir); isDir {
				watcher.watchDirectory(subDir)
			} else if err != nil {
				log.Errorf("%s", err)
			}
		}
	}
}

func (watcher *Watcher) watchFile(path string) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}

	if stat.IsDir() {
		log.Critical("watchFile called on a directory")
		return
	}

	if err = watcher.fsWatcher.Add(path); err != nil {
		log.Errorf("Failed to watch %s: %s", path, err)
	} else {
		log.Infof("Watching %s", path)
	}
}
