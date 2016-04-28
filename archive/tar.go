package archive

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/charleswong/scraper/util"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	IdStep      = 1000
	ArchiveSize = IdStep * 10
)

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
	if err != nil {
		log.Println(err)
		return err
	}

	defer tarfile.Close()
	var fileWriter io.WriteCloser = tarfile

	if strings.HasSuffix(desFile, ".gz") {
		fileWriter = gzip.NewWriter(tarfile) // add a gzip filter
		defer fileWriter.Close()             // if user add .gz in the destination filename
	}

	tarfileWriter := tar.NewWriter(fileWriter)
	defer tarfileWriter.Close()

	walkFn := func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}
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
	log.Printf("Archived Id from %d to %d in %s.\n", startId, endId-1, basePath)
	deleteRange(startId, endId, basePath)
	return nil
}

func Archive(startId, endId int, desFolder, subDesFolder, basePath string) error {
	os.MkdirAll(desFolder, 0666)
	if util.IsLowDiskSpace() {
		return errors.New("Low disk space")
	}
	desFile := fmt.Sprintf("%s/%s/%d-%d.tar.gz", desFolder, subDesFolder, startId, endId-1)
	err := archiveRange(startId, endId, desFile, basePath)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
