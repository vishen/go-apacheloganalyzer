package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	// data       []Information
	search_for     []string
	found_searches map[string]int

	mutex *sync.Mutex
}

func NewStatistics(search_for []string) Statistics {
	s := Statistics{search_for: search_for}
	s.found_searches = make(map[string]int, len(search_for))
	s.mutex = &sync.Mutex{}
	return s
}

func (s *Statistics) addInformation(info Information) {
	// s.data = append(s.data, info)

	for _, sf := range s.search_for {
		if strings.Contains(info.path, sf) {
			s.mutex.Lock()
			f := s.found_searches[sf]
			f += 1
			s.found_searches[sf] = f
			s.mutex.Unlock()
		}
	}
}

func (s *Statistics) pathCount(path string) int {
	// total := 0
	// for _, info := range s.data {
	// 	if strings.Contains(info.path, path) {
	// 		total += 1
	// 	}
	// }

	// return total

	return s.found_searches[path]
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
			// log.Printf("Found associated file: %s - analyzing...", fullpath)

		}
	}

	log.Println(found_files)
	// Add a WaitGroup so we can run each file
	// asyncrounously
	var wg sync.WaitGroup
	wg.Add(len(found_files))
	for _, fullpath := range found_files {
		go func(filename string) {
			analyzeFile(filename)
			wg.Done()
		}(fullpath)
	}
	wg.Wait()
	return found_files
}

func splitWithPosition(s, sep string, position int) string {
	splitted := strings.Split(s, sep)
	if (len(splitted) - 1) < position {
		return ""
	}

	return splitted[position]
}

func _analyzeFile(file_reader io.Reader) {

	r := bufio.NewReader(file_reader)
	var url, path, ipaddress, line, http_status_code string
	var info Information
	// var _line []byte
	// var err error.Error
	for {
		_line, _, err := r.ReadLine()
		// log.Println(_line)
		if err != nil {
			break
		}
		line = string(_line)
		if line == "" {
			continue
		}

		url = splitWithPosition(line, "\"", 1)
		path = splitWithPosition(url, " ", 1)
		ipaddress = splitWithPosition(line, " ", 0)

		// Only allow requests that returned a 200
		http_status_code = splitWithPosition(line, " ", 8)
		if http_status_code != "200" {
			// log.Println("Ignoring:", http_status_code)
			continue
		}
		info = Information{url: url, path: path, ipaddress: ipaddress}
		// log.Println(info)
		statistics.addInformation(info)
	}
}

func analyzeFile(filename string) {
	// var file_content []byte
	switch filepath.Ext(filename) {
	case ".gz":
		log.Println("[Error] Found gzip file - please unzip files.")
	default:
		// file_content, err := ioutil.ReadFile(filename)
		file, err := os.Open(filename)
		log.Println(filename)
		if err != nil {
			log.Printf("[Error] %s\n", err)
		} else {
			_analyzeFile(file)
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

	statistics = NewStatistics(search_for)

	log.Printf("Root folder: %s\n", root_folder)
	log.Printf("Log Type: %s\n", log_type)

	_ = findFiles(root_folder)

	for _, sf := range search_for {
		if sf == "" {
			continue
		}
		log.Printf("%s: %d", sf, statistics.pathCount(sf))
	}

}
