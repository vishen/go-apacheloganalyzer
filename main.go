package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var root_folder string
var log_type string
var statistics Statistics

const (
	DATE_FORMAT string = "02/Jan/2006"
)

// Define us a type so we can sort it
type TimeSlice []time.Time

// Forward request for length
func (p TimeSlice) Len() int {
	return len(p)
}

// Define compare
func (p TimeSlice) Less(i, j int) bool {
	return p[i].Before(p[j])
}

// Define swap over an array
func (p TimeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type Information struct {
	url            string
	path           string
	ipaddress      string
	forwarded_from string
	date_string    string
}

type FoundSearch struct {
	path       string
	date_count map[time.Time]int
	mutex      *sync.Mutex
}

func NewFoundSearch(path string) *FoundSearch {
	return &FoundSearch{path: path, mutex: &sync.Mutex{}, date_count: make(map[time.Time]int)}
}

func (fs *FoundSearch) incr(date_string string) {
	t, err := time.Parse(DATE_FORMAT, date_string)
	if err != nil {
		log.Fatal(err)
	}
	fs.mutex.Lock()
	count, _ := fs.date_count[t]
	count += 1
	fs.date_count[t] = count
	fs.mutex.Unlock()
}

type Statistics struct {
	// data       []Information
	forwarded_from string
	search_for     []string
	found_searches map[string]*FoundSearch

	total_count int
}

func NewStatistics(search_for []string, forwarded_from string) Statistics {
	s := Statistics{search_for: search_for, forwarded_from: forwarded_from, total_count: 0}
	s.found_searches = make(map[string]*FoundSearch, len(search_for))
	// s.mutex = &sync.Mutex{}
	return s
}

func (s *Statistics) addInformation(info Information) {
	// s.data = append(s.data, info)

	if s.forwarded_from != "" && !strings.Contains(info.forwarded_from, s.forwarded_from) {
		return
	}

	for _, sf := range s.search_for {
		if strings.Contains(info.path, sf) {
			var f *FoundSearch
			var ok bool
			f, ok = s.found_searches[sf]
			if !ok {
				f = NewFoundSearch(sf)
				s.found_searches[sf] = f
			}
			f.incr(info.date_string)
		}
	}
}

func (s *Statistics) printPathCount(path string) {

	for _, fs := range s.found_searches {

		var keys TimeSlice
		for k := range fs.date_count {
			keys = append(keys, k)
		}

		sort.Sort(keys)
		for _, date := range keys {
			count, _ := fs.date_count[date]
			s.total_count += count
			fmt.Printf("%s: [%s] %d\n", path, date.Format(DATE_FORMAT), count)
		}
	}
	return
}

func (s *Statistics) printTotal() {
	fmt.Println("Total: ", s.total_count)
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
	var url, path, ipaddress, line, http_status_code, forwarded_from, date_string string
	var info Information
	// var _line []byte
	// var err error.Error
	// var wg sync.WaitGroup
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
		forwarded_from = splitWithPosition(line, " ", 10)
		if http_status_code != "200" {
			// log.Println("Ignoring:", http_status_code)
			continue
		}
		date_string = splitWithPosition(line, "[", 1)[0:11]
		info = Information{url: url, path: path, ipaddress: ipaddress,
			forwarded_from: forwarded_from, date_string: date_string}
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
	forwarded_from := flag.String("forwarded_from", "", "")

	flag.Parse()

	search_for := strings.Split(*_search_for, ",")

	statistics = NewStatistics(search_for, *forwarded_from)

	log.Printf("Root folder: %s\n", root_folder)
	log.Printf("Log Type: %s\n", log_type)

	_ = findFiles(root_folder)

	for _, sf := range search_for {
		if sf == "" {
			continue
		}
		statistics.printPathCount(sf)
		// log.Printf("%s: %d", sf, statistics.pathCount(sf))
	}

	statistics.printTotal()

}
