package config

import (
	"github.com/charleswong/scraper/model"
	"testing"
)

func TestConfig(t *testing.T) {
	configFile = "scraper.conf"
	config = &model.ScraperConfig{}
	config.TaskFile = "jiayuan.task"
	config.Proxies = []string{"localhost:1081", "localhost:1082"}
	config.ThreadNum = 1
	config.ValidImgNum = 2
	config.DataFolder = "deepavatar/"
	config.ArchiveFolder = "avatar_tars/"
	config.TmpFolder = "deep_tmp/"

	jiayuanTask := &model.JiayuanTask{
		Type: model.TaskType_JIAYUAN,
		IdProfileTask: &model.IdProfileTask{
			BeginId: 100,
			EndId:   200,
		},
	}
	tasks = []model.Task{jiayuanTask}

	SaveTask()
	SaveConfig()
}
