package main

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/y-yagi/rnotify"
)

type Runner struct {
	watcher *rnotify.Watcher
	eventCh chan string
	cfg     Config
}

type Config struct {
	Actions []Action
}

type Action struct {
	Command       string
	Extensions    []string
	extensionsMap map[string]bool
}

func (r *Runner) Run() error {
	done := make(chan bool)

	if err := r.watch(); err != nil {
		return err
	}

	for {
		var filename string

		select {
		case filename = <-r.eventCh:
			time.Sleep(500 * time.Millisecond)
			r.discardEvents()

			for _, action := range r.cfg.Actions {
				if _, ok := action.extensionsMap[filepath.Ext(filename)]; ok {
					fmt.Printf("Run command\n")
					stdoutStderr, err := exec.Command("go", "build", ".").CombinedOutput()
					if err != nil {
						fmt.Printf("command failed: %v\n", err)
					}

					if len(string(stdoutStderr)) != 0 {
						fmt.Printf("%s\n", stdoutStderr)
					}
				}
			}
		}
	}

	<-done
	return nil
}

func (r *Runner) watch() error {
	if err := r.watcher.Add("."); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-r.watcher.Events:
				fmt.Printf("%v\n", event)
				if !ok {
					return
				}

				r.eventCh <- event.Name
			case err, ok := <-r.watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	return nil
}

func (r *Runner) discardEvents() {
	for {
		select {
		case <-r.eventCh:
		default:
			return
		}
	}
}

func main() {
	// cmd.Env = append(os.Environ(), command.envs...)

	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	watcher, err := rnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	r := Runner{
		eventCh: make(chan string, 1000),
		watcher: watcher,
		cfg:     *cfg,
	}

	defer r.watcher.Close()

	if err = r.Run(); err != nil {
		log.Fatal(err)
	}
}

func parseConfig() (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile("porter.toml", &cfg); err != nil {
		return nil, err
	}

	for k, action := range cfg.Actions {
		m := map[string]bool{}
		for _, extension := range action.Extensions {
			m[extension] = true
		}
		cfg.Actions[k].extensionsMap = m
	}

	return &cfg, nil
}
