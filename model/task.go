package model

import (
	"encoding/json"
	"errors"
	"log"
)

type Task interface {
	GetType() TaskType
	GetIdProfileTask() *IdProfileTask
}

func (t *SocialImageTask) GetType() TaskType {
	return t.Type
}

func UnpackScraperTask(scraperTask *ScraperTask) (Task, error) {
	switch scraperTask.Type {
	case TaskType_JIAYUAN:
		t := &SocialImageTask{}
		err := json.Unmarshal([]byte(scraperTask.Data), t)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return t, nil
	case TaskType_BAIHE:
		t := &SocialImageTask{}
		err := json.Unmarshal([]byte(scraperTask.Data), t)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return t, nil
	case TaskType_RENREN:
		t := &SocialImageTask{}
		err := json.Unmarshal([]byte(scraperTask.Data), t)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return t, nil
	}
	return nil, errors.New("Invalid task type to unpack.")

}

func PackScraperTask(t Task) (*ScraperTask, error) {
	bytes, err := json.Marshal(t)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &ScraperTask{
		Type: t.GetType(),
		Data: string(bytes),
	}, nil
}
