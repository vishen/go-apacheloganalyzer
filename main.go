package main

import (
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

var root_folder string
var log_type string
var statistics Statistics

type Information struct {
	url       string
	path      string
	ipaddress string
}

type Statistics struct {
	data []Information
}

func (s *Statistics) addInformation(info Information) {
	s.data = append(s.data, info)
}

func (s *Statistics) pathCount(path string) int {
	total := 0
	for _, info := range s.data {
		if strings.Contains(info.path, path) {
			total += 1
		}
	}

	return total
}

func findFiles(dir string) []string {
	found_files := []string{}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		log.Printf("%s\n", f.Name())

		if strings.Contains(f.Name(), log_type) {
			fullpath := filepath.Join(root_folder, f.Name())
			found_files = append(found_files, fullpath)
			log.Printf("Found associated file: %s", fullpath)
			analyzeFile(fullpath)
		}
	}

	return found_files
}

func splitWithPosition(s, sep string, position int) string {
	splitted := strings.Split(s, sep)
	if (len(splitted) - 1) < position {
		return ""
	}

	return splitted[position]
}

func _analyzeFile(file_content []byte) {

	// log.Printf("%s\n", file_content)
	for _, line := range strings.Split(string(file_content), "\n") {
		if line == "" {
			continue
		}

		url := splitWithPosition(line, "\"", 1)
		path := splitWithPosition(url, " ", 1)
		ipaddress := splitWithPosition(line, " ", 0)

		info := Information{url: url, path: path, ipaddress: ipaddress}
		log.Println(info)
		statistics.addInformation(info)
	}
}

func analyzeFile(filename string) {
	// var file_content []byte
	switch filepath.Ext(filename) {
	case ".gz":
		log.Fatal("[Error] Found gzip file - please unzip files.")
	default:
		file_content, err := ioutil.ReadFile(filename)

		if err != nil {
			log.Printf("[Error] %s\n", err)
		} else {
			_analyzeFile(file_content)
		}
	}
}

func init() {
	flag.StringVar(&root_folder, "root_folder", "", "The root folder to search for logs.")
	flag.StringVar(&log_type, "log_type", "access", "The type of log files to read. Defaults to `access`")
}

func main() {

	_search_for := flag.String("search_for", "", "")

	flag.Parse()

	search_for := strings.Split(*_search_for, ",")

	log.Printf("Root folder: %s\n", root_folder)
	log.Printf("Log Type: %s\n", log_type)

	statistics = Statistics{}

	_ = findFiles(root_folder)

	for _, sf := range search_for {
		if sf == "" {
			continue
		}
		log.Printf("%s: %d", sf, statistics.pathCount(sf))
	}

}
