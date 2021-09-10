package main

import (
	"compress/flate"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mholt/archiver/v3"
)

var zip archiver.Zip = archiver.Zip{
	OverwriteExisting:      true,
	MkdirAll:               true,
	CompressionLevel:       flate.DefaultCompression,
	ContinueOnError:        true,
	ImplicitTopLevelFolder: false,
}

func newFilePath(prefix, postfix string) (string, int) {
	fileId := 1
	for {
		if _, err := os.Stat(prefix + strconv.Itoa(fileId) + postfix); os.IsNotExist(err) {
			log.Println(fileId)
			return prefix + strconv.Itoa(fileId) + postfix, fileId
		}
		fileId += 1
	}
}

func ListFileIDs(dirname, fileext string) []int {
	ret := sort.IntSlice{}
	files, err := os.ReadDir(dirname)
	if err != nil {
		log.Fatal(err)
	}
	for _, info := range files {
		if !info.IsDir() {
			name := strings.Trim(info.Name(), fileext)
			if id, err := strconv.Atoi(name); err == nil {
				ret = append(ret, id)
			}
		}
	}
	ret.Sort()
	return ret
}

func ListRecordings() []int {
	return ListFileIDs("/recordings", ".json")
}

func ListVolumesSnapshots() []int {
	return ListFileIDs("/snapshots", ".zip")
}

func (p *Proxy) writeRecording() (int, error) {
	filename, fileId := newFilePath("/recordings/", ".json")
	p.lastSavedId = fileId
	bytes, err := json.MarshalIndent(p.recording, "", " ")
	if err != nil {
		return 0, err
	}
	return fileId, os.WriteFile(filename, bytes, 0b_110110110)
}

func loadRecording(id string) (*Recording, error) {
	filepath := "/recordings/" + id + ".json"
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	recording := &Recording{}
	err = json.Unmarshal(bytes, recording)
	if err != nil {
		return nil, err
	}
	return recording, nil
}

func writeVolumes() (int, error) {
	filename, fileId := newFilePath("/snapshots/", ".zip")
	err := zip.Archive([]string{"/volumes"}, filename)
	if err != nil {
		log.Println(err)
	}
	return fileId, err
}

func loadVolumes(id string) error {
	if id == "" {
		return errors.New("No Volumes Snapshot")
	}
	filepath := "/snapshots/" + id + ".zip"
	err := zip.Unarchive(filepath, "/volumes/..")
	if err != nil {
		log.Println(err)
	}
	return err
}

func latestVolumes() string {
	files, _ := os.ReadDir("/snapshots")
	latest := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := strings.Trim(file.Name(), ".zip")
		if id, err := strconv.Atoi(name); err == nil {
			if id > latest {
				latest = id
			}
		}
	}
	return strconv.Itoa(latest)
}
