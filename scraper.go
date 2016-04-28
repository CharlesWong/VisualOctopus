package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/charleswong/scraper/archive"
	"github.com/charleswong/scraper/config"
	"github.com/charleswong/scraper/model"
	"github.com/charleswong/scraper/util"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	basePath = "./deepavatar/"
)

func downloadFile(url, path string) error {
	res, err := http.Get(url)
	for i := 0; ; i++ {
		if err != nil {
			log.Printf("Error: http.Get -> %v\n", err)
			if i > 10 {
				return err
			} else {
				res, err = http.Get(url)
			}
		} else {
			break
		}
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error: ioutil.ReadAll -> %v\n", err)
		return err
	}

	err = ioutil.WriteFile(path, data, 0666)
	if err != nil {
		log.Printf("Error: ioutil.WriteAll -> %v\n", err)
		return err
	}

	log.Printf("Downloaded %d bytes from %s -> %s\n", len(data), url, path)
	return nil
}

func appendFile(line, path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Error: ioutil.WriteAll -> %v\n", err)
		return err
	}
	defer file.Close()
	file.WriteString(line + "\n")

	log.Printf("Downloaded %d bytes from %s -> %s\n", len(line), line, path)
	return nil
}

func saveBytes(data []byte, path string) error {
	ioutil.WriteFile(path, data, 0666)

	log.Printf("Downloaded %d bytes Data -> %s\n", len(data), path)
	return nil
}

func getPath(id int, taskType model.TaskType) string {
	pathes := []string{
		basePath,
		SiteName[taskType],
		strconv.Itoa(id / 1000000),
		strconv.Itoa(id / 1000 % 1000),
		strconv.Itoa(id % 1000),
	}
	p := path.Join(pathes...)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		// dir does not exist
		err := os.MkdirAll(p, 0777)
		if err != nil {
			return ""
		}
	}
	return p
}

func getProfilePath(id int, taskType model.TaskType) string {
	pathes := []string{
		getPath(id, taskType),
		strconv.Itoa(id) + ".html",
	}
	return path.Join(pathes...)
}

func getImagePath(id int, url string, taskType model.TaskType) string {
	urlTokens := strings.Split(url, "/")
	if l := len(urlTokens); l == 0 {
		return ""
	}
	imageName := urlTokens[len(urlTokens)-1]
	pathes := []string{
		getPath(id, taskType),
		imageName,
	}
	return path.Join(pathes...)
}

func getStatPath(id int, taskType model.TaskType) string {
	pathes := []string{
		basePath,
		SiteName[taskType],
		"stats",
	}
	p := path.Join(pathes...)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		// dir does not exist
		err := os.MkdirAll(p, 0777)
		if err != nil {
			return ""
		}
	}

	pathes = []string{
		p,
		strconv.Itoa(id/1000) + ".stats",
	}
	return path.Join(pathes...)
}

func saveProfilePage(id int, url string, taskType model.TaskType) error {
	p := getProfilePath(id, taskType)
	err := downloadFile(url, p)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func saveImage(id int, url string, taskType model.TaskType) error {
	p := getImagePath(id, url, taskType)
	err := downloadFile(url, p)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func saveImageAsync(id int, url string, taskType model.TaskType, chFinished chan int) error {
	go func() {
		defer func() {
			chFinished <- 1
		}()
		p := getImagePath(id, url, taskType)
		err := downloadFile(url, p)
		if err != nil {
			log.Println(err)
		}

	}()
	return nil
}

var (
	statsLock = &sync.Mutex{}
)

func saveStats(p *Profile, taskType model.TaskType) error {
	statsLock.Lock()
	defer statsLock.Unlock()
	path := getStatPath(p.Id, taskType)
	err := appendFile(p.ToString(), path)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

type Profile struct {
	Id        int
	ImageURLs []string
	RawData   []byte
}

func NewProfile() *Profile {
	return &Profile{
		ImageURLs: make([]string, 0),
	}
}

func (p *Profile) ToString() string {
	parts := []string{
		strconv.Itoa(p.Id),
		strconv.Itoa(len(p.ImageURLs)),
	}
	return strings.Join(parts, "\t")
}

var (
	ImageUrlValidationFuncs = make(map[model.TaskType]func(*html.Node) []string)
	ProfileTemplate         = make(map[model.TaskType]string)
	SiteName                = make(map[model.TaskType]string)
)

func InitMaps() {
	ProfileTemplate[model.TaskType_JIAYUAN] = "http://www.jiayuan.com/%d"
	ProfileTemplate[model.TaskType_BAIHE] = "http://profile1.baihe.com/?oppId=%d"
	ImageUrlValidationFuncs[model.TaskType_JIAYUAN] = ExtractJiayuanImage
	ImageUrlValidationFuncs[model.TaskType_BAIHE] = ExtractBaiheImage
	SiteName[model.TaskType_JIAYUAN] = "Jiayuan"
	SiteName[model.TaskType_BAIHE] = "Baihe"
}

func ExtractJiayuanImage(n *html.Node) []string {
	images := make([]string, 0)
	if n.Type == html.ElementNode && n.Data == "img" {
		valid := false
		imgUrl := ""
		for _, attr := range n.Attr {
			if attr.Key == "_src" {
				imgUrl = attr.Val
			}
			if attr.Key == "class" {
				if attr.Val == "img_absolute" {
					valid = true
				} else {
					break
				}
			}
		}
		if valid {

			if !strings.Contains(imgUrl, "photo_invite_") && !strings.Contains(imgUrl, "_bp.jpg") && !strings.Contains(imgUrl, "avatar_p.jpg") && !strings.Contains(imgUrl, "_p.jpg") {
				log.Println("Found image: ", imgUrl)
				images = append(images, imgUrl)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		images = append(images, ExtractJiayuanImage(c)...)
	}
	return images
}

func ExtractBaiheImage(n *html.Node) []string {
	images := make([]string, 0)
	if n.Type == html.ElementNode && n.Data == "img" {
		valid := false
		imgUrl := ""
		for _, attr := range n.Attr {
			if attr.Key == "src" {
				imgUrl = attr.Val
				valid = true
				break
			}
		}
		if valid {
			if strings.Contains(imgUrl, "/290_290/") && strings.Contains(imgUrl, ".jpg") {
				log.Println("Found image: ", imgUrl)
				images = append(images, imgUrl)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		images = append(images, ExtractBaiheImage(c)...)
	}
	return images
}

func parseProfile(id int, b []byte, taskType model.TaskType) (*Profile, error) {
	reader := bytes.NewReader(b)
	doc, err := html.Parse(reader)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	profile := NewProfile()
	profile.Id = id
	profile.RawData = b
	profile.ImageURLs = ImageUrlValidationFuncs[taskType](doc)
	return profile, nil
}

var (
	client = &http.Client{}
)

func crawl(id int, url string, taskType model.TaskType) (*Profile, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")

	resp, err := client.Do(req)
	for i := 0; i < 10; i++ {
		if err != nil {
			log.Println("Crawler Error: Failed to crawl \"" + url + "\"")
			resp, err = client.Do(req)
		} else {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	b := resp.Body
	defer b.Close()

	data, err := ioutil.ReadAll(b)
	if err != nil {
		log.Println("Crawler Error: ioutil.ReadAll -> %v", err)
		return nil, err
	}

	profile, err := parseProfile(id, data, taskType)

	return profile, err
}

func save(profile *Profile, taskType model.TaskType) error {

	// Save images.
	if len(profile.ImageURLs) > 0 {
		chImg := make(chan int, len(profile.ImageURLs))
		for _, imgUrl := range profile.ImageURLs {
			// saveImage(profile.Id, imgUrl)
			saveImageAsync(profile.Id, imgUrl, taskType, chImg)
		}

		finishedImg := 0
		for finishedImg < len(profile.ImageURLs) {
			select {
			case <-chImg:
				finishedImg++
			}
		}
	}
	// Save profile page.
	saveBytes(profile.RawData, getProfilePath(profile.Id, taskType))
	saveStats(profile, taskType)

	return nil
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	configFile := flag.String("config", "scraper.conf", "Config file.")
	// archiveBefore := flag.Bool("archive_before", false, "Archive previous range.")
	flag.Parse()

	config.ConfigFile = *configFile
	c := config.GetConfig()
	tasks := config.GetTasks()

	if len(tasks) == 0 {
		log.Fatal("No task to run.")
	}

	if len(c.Proxies) == 0 {
		log.Println("No proxies set")
	} else {
		os.Setenv("HTTP_PROXY", c.Proxies[0])
		os.Setenv("HTTPS_PROXY", c.Proxies[0])
	}

	InitMaps()
	// if *archiveBefore {
	// 	log.Println("Archive previous range.")
	// 	id := int(tasks[0].GetIdProfileTask().BeginId)
	// 	id = id - id%archive.ArchiveSize

	// 	err := archive.Archive(id-archive.ArchiveSize, id, c.ArchiveFolder, c.DataFolder)
	// 	if err != nil {
	// 		log.Println(err)
	// 	} else {
	// 		log.Println("Archived previous package.")
	// 	}
	// }
	chTask := make(chan int, c.ThreadNum)
	for _, task := range tasks {
		go func() {
			for id := task.GetIdProfileTask().BeginId - 1; id < task.GetIdProfileTask().EndId && !util.IsLowDiskSpace(); id++ {
				chTask <- 1
				taskId := int(id)
				go func() {
					defer func() {
						<-chTask
					}()
					log.Println("Crawling Id: ", taskId)
					url := fmt.Sprintf(ProfileTemplate[task.GetType()], taskId)
					profile, err := crawl(int(taskId), url, task.GetType())
					if err != nil {
						log.Println(err)
						return
					}
					if profile != nil && len(profile.ImageURLs) >= int(c.ValidImgNum) {
						save(profile, task.GetType())
					}
				}()
				if int(taskId+1)%archive.ArchiveSize == 0 {
					err := archive.Archive(taskId+1-archive.ArchiveSize, taskId+1, c.ArchiveFolder, SiteName[task.GetType()], path.Join(c.DataFolder, SiteName[task.GetType()]))
					if err != nil {
						log.Println(err)
					}
				}
				task.GetIdProfileTask().BeginId = int64(taskId)
				err := config.SaveTasks()
				if err != nil {
					log.Println(err)
				}
			}
		}()
	}

	// Handle exiting signals and process.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			err := config.SaveTasks()
			if err != nil {
				log.Println(err)
			}
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
