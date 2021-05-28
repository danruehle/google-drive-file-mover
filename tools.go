package main

import (
	"fmt"
	"hash/crc32"
	"io"
	"log"

	"google.golang.org/api/drive/v3"
)

func GetRootId(srv *drive.Service) string {
	r, err := srv.Files.Get("root").Fields("id, name").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	fmt.Printf("root: %s (%s)\n", r.Name, r.Id)
	return r.Id
}

func GetRootFilesId(srv *drive.Service) string {
	r, err := srv.Files.List().Q("name = 'RootFiles'").PageSize(2).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve RootFiles: %v", err)
	}
	if len(r.Files) != 1 {
		log.Fatalf("Invalid number of RootFiles returned: %#v", r.Files)
	}
	f := r.Files[0]
	fmt.Printf("RootFiles: %s (%s)\n", f.Name, f.Id)
	return f.Id
}

func ShouldDeleteFile(name string, percent int) bool {
	h := crc32.NewIEEE()
	io.WriteString(h, name)
	r := h.Sum32()
	return int(r%100) < percent
}
