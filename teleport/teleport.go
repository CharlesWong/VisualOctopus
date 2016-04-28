package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	// "golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var (
	FreeDiskThreshold = 1024 * 1024 * 1024
)

func GetFreeDisk() int {
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
		return 0
	}
	syscall.Statfs(wd, &stat)
	// Available blocks * size per block = available space in bytes
	return int(stat.Bavail * uint64(stat.Bsize))
}

func IsLowDiskSpace() bool {
	return GetFreeDisk() < FreeDiskThreshold
}

func checkerror(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	IdStep = 1000
)

const privateKey = `content of id_rsa`

// func teleportTar(desFile string, remoteServer string) error {
// 	signer, _ := ssh.ParsePrivateKey([]byte(privateKey))
// 	clientConfig := &ssh.ClientConfig{
// 		User: "jedy",
// 		Auth: []ssh.AuthMethod{
// 			ssh.PublicKeys(signer),
// 		},
// 	}
// 	client, err := ssh.Dial("tcp", "127.0.0.1:22", clientConfig)
// 	if err != nil {
// 		panic("Failed to dial: " + err.Error())
// 	}
// 	session, err := client.NewSession()
// 	if err != nil {
// 		panic("Failed to create session: " + err.Error())
// 	}
// 	defer session.Close()
// 	go func() {
// 		w, _ := session.StdinPipe()
// 		defer w.Close()
// 		content := "123456789\n"
// 		fmt.Fprintln(w, "D0755", 0, "testdir") // mkdir
// 		fmt.Fprintln(w, "C0644", len(content), "testfile1")
// 		fmt.Fprint(w, content)
// 		fmt.Fprint(w, "\x00") // transfer end with \x00
// 		fmt.Fprintln(w, "C0644", len(content), "testfile2")
// 		fmt.Fprint(w, content)
// 		fmt.Fprint(w, "\x00")
// 	}()
// 	if err := session.Run("/usr/bin/scp -tr ./"); err != nil {
// 		panic("Failed to run: " + err.Error())
// 	}
// }

func deleteRange(startId, endId int, basePath string) error {
	for i := startId; i < endId; i += IdStep {
		pathes := []string{
			basePath,
			strconv.Itoa(i / 1000000),
			strconv.Itoa(i / 1000 % 1000),
		}
		p := path.Join(pathes...)

		err := os.RemoveAll(p)
		if err != nil {
			return err
		}
	}
	log.Printf("Deleted files of Id from %d to %d.\n", startId, endId-1)
	return nil
}

func archiveRange(startId, endId int, desFile string, basePath string) error {
	os.Remove(desFile)
	tarfile, err := os.Create(desFile)
	checkerror(err)

	defer tarfile.Close()
	var fileWriter io.WriteCloser = tarfile

	if strings.HasSuffix(desFile, ".gz") {
		fileWriter = gzip.NewWriter(tarfile) // add a gzip filter
		defer fileWriter.Close()             // if user add .gz in the destination filename
	}

	tarfileWriter := tar.NewWriter(fileWriter)
	defer tarfileWriter.Close()

	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.Mode().IsDir() {
			return nil
		}
		// Because of scoping we can reference the external root_directory variable
		new_path := path[len(basePath):]
		// log.Println("base_path: ", basePath, path)
		// log.Println("new_path: ", new_path)
		if len(new_path) == 0 {
			return nil
		}
		fr, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fr.Close()

		if h, err := tar.FileInfoHeader(info, new_path); err != nil {
			log.Fatalln(err)
		} else {
			h.Name = new_path
			if err = tarfileWriter.WriteHeader(h); err != nil {
				log.Fatalln(err)
			}
		}
		if _, err := io.Copy(tarfileWriter, fr); err != nil {
			log.Fatalln(err)
		} else {
			// log.Println(length)
		}
		return nil
	}

	for i := startId; i < endId; i += IdStep {
		pathes := []string{
			basePath,
			strconv.Itoa(i / 1000000),
			strconv.Itoa(i / 1000 % 1000),
		}
		p := path.Join(pathes...)

		if err = filepath.Walk(p, walkFn); err != nil {
			return err
		}
	}
	log.Printf("Archived Id from %d to %d.\n", startId, endId-1)
	deleteRange(startId, endId, basePath)
	return nil
}

type TeleportStatus struct {
	LastEndId    int
	StopId       int
	Destination  string
	BasePath     string
	RemoteServer string
}

var (
	statusPath = "./teleport.status"
)

func readStatus() (*TeleportStatus, error) {
	bytes, err := ioutil.ReadFile(statusPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	s := &TeleportStatus{}
	err = json.Unmarshal(bytes, s)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return s, nil
}

func saveStatus(s *TeleportStatus) error {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(statusPath, bytes, 0666)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func archiveRoutine() error {
	lastEndId := 0

	basePath := ""
	status, err := readStatus()

	if status != nil {
		lastEndId = status.LastEndId
		basePath = status.BasePath
	} else {
		log.Fatal("Invalid Id range to teleport.")
		return errors.New("Invalid Id range to teleport.")
	}
	os.MkdirAll(status.Destination, 0666)
	currentId := getCurrentId(basePath)
	if status.StopId > 0 {
		currentId = status.StopId
	}
	for id := lastEndId; id+IdStep*10 <= currentId; id += IdStep * 10 {
		if IsLowDiskSpace() {
			break
		}
		desFile := fmt.Sprintf("%s/%d-%d.tar.gz", status.Destination, id, id+IdStep*10-1)
		err := archiveRange(id, id+IdStep*10, desFile, basePath)
		if err != nil {
			return err
		}
		status.LastEndId = id + IdStep*10
		err = saveStatus(status)
		if err != nil {
			return err
		}
	}
	err = saveStatus(status)
	if err != nil {
		return err
	}
	return nil
}

func getCurrentFolder(basePath string) (string, int) {
	files, _ := ioutil.ReadDir(basePath)
	maxId := math.MinInt64
	maxFolder := ""

	for _, f := range files {
		pathes := strings.Split(f.Name(), "/")
		if len(pathes) > 0 {
			t, err := strconv.Atoi(pathes[len(pathes)-1])
			if err != nil {
				continue
			}
			if t > maxId {
				maxId = t
				maxFolder = f.Name()
			}
		}
	}

	if len(maxFolder) > 0 {
		return path.Join(basePath, maxFolder), maxId
	}
	return "", 0
}

func getCurrentId(basePath string) int {
	folder1, id1 := getCurrentFolder(basePath)
	_, id2 := getCurrentFolder(folder1)
	return id1*1000000 + id2*1000
}

func main() {
	flag.Parse() // get the arguments from command line
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	archiveRoutine()
}
