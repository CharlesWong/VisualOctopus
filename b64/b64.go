package main

import (
	"encoding/base64"
	"flag"
	"io/ioutil"
	"log"
)

func main() {
	f := flag.String("file", "", "File to convert")
	flag.Parse()

	log.SetFlags(log.Lshortfile | log.LstdFlags)
	bytes, err := ioutil.ReadFile(*f)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(bytes))
	ioutil.WriteFile("test.jpg", decoded, 066)

}
