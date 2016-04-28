package config

import (
	"encoding/json"
	"github.com/charleswong/scraper/model"
	"io/ioutil"
	"log"
	"sync"
)

var (
	ConfigFile string
	config     *model.ScraperConfig
	tasks      []model.Task
	configLock = &sync.Mutex{}
)

func GetConfig() *model.ScraperConfig {
	if config == nil {
		configLock.Lock()
		defer configLock.Unlock()
		if config == nil {
			loadConfig()
		}
	}
	return config
}

func GetTasks() []model.Task {
	if tasks == nil {
		configLock.Lock()
		defer configLock.Unlock()
		if tasks == nil {
			loadTasks()
		}
	}
	return tasks
}

func loadTasks() error {
	c := GetConfig()
	if c == nil || len(c.TaskFile) == 0 {
		log.Fatal("No task file.")
	}
	bytes, err := ioutil.ReadFile(c.TaskFile)
	if err != nil {
		log.Println(err)
		return err
	}
	scraperTasks := &model.ScraperTasks{}
	err = json.Unmarshal(bytes, scraperTasks)
	if err != nil {
		log.Println(err)
		return err
	}

	tasks = make([]model.Task, 0)
	for _, t := range scraperTasks.Tasks {
		task, err := model.UnpackScraperTask(t)
		if err != nil {
			log.Println(err)
			return err
		}
		tasks = append(tasks, task)
	}
	return nil
}

func SaveTasks() error {
	c := GetConfig()
	if c == nil || len(c.TaskFile) == 0 {
		log.Fatal("No task file.")
	}
	scraperTasks := &model.ScraperTasks{}

	for _, t := range tasks {
		packedTask, err := model.PackScraperTask(t)
		if err != nil {
			log.Println(err)
			return err
		}
		scraperTasks.Tasks = append(
			scraperTasks.Tasks,
			packedTask,
		)
	}

	bytes, err := json.Marshal(scraperTasks)
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(c.TaskFile, bytes, 0666)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("Task saved to ", c.TaskFile)
	return nil
}

func loadConfig() error {
	if len(ConfigFile) == 0 {
		log.Fatal("No config file.")
	}
	bytes, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		log.Println(err)
		return err
	}
	config = &model.ScraperConfig{}
	err = json.Unmarshal(bytes, config)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func SaveConfig() error {
	bytes, err := json.Marshal(GetConfig())
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(ConfigFile, bytes, 0666)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
