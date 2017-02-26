package main

import (
	"github.com/go-fsnotify/fsnotify"
	// "io"
	"io/ioutil"
	"log"
	"os"
	pathUtil "path"
)

type Watcher struct {
	fsWatcher *fsnotify.Watcher

	Updates    chan string
	cancelChan chan bool
}

func NewWatcher(path string, cancel chan bool) (watcher *Watcher, err error) {
	watcher = &Watcher{}
	watcher.fsWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher.Updates = make(chan string)
	watcher.cancelChan = cancel
	watcher.watchDirectory(path)

	return watcher, nil
}

func (watcher *Watcher) Run() {
	for {
		select {
		case event := <-watcher.fsWatcher.Events:
			log.Println(event)
			watcher.Updates <- event.Name
		case err := <-watcher.fsWatcher.Errors:
			log.Println(err)
		case <-watcher.cancelChan:
			if err := watcher.fsWatcher.Close(); err != nil {
				log.Println(err)
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
		// log.Printf("ReadDir failed for %s: %s", pathStr, err)
		return
	}

	if err = watcher.fsWatcher.Add(path); err != nil {
		log.Fatalf("Failed to watch %s: %s", err)
	} else {
		log.Printf("Watching %s", path)

		for _, entry := range entries {
			subDir := pathUtil.Join(path, entry.Name())
			if isDir, err := isDirectory(subDir); isDir {
				watcher.watchDirectory(subDir)
			} else if err != nil {
				log.Fatalf("%s", err)
			}
		}
	}
}

func watchFile(path string) {

}
