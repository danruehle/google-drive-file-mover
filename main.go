package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	moveFiles(ctx, srv)
	//deleteFiles(ctx, srv)
}

func moveFiles(ctx context.Context, srv *drive.Service) {
	rootId := GetRootId(srv)
	rootFilesId := GetRootFilesId(srv)

	fmt.Println("Files:")

	pageSize := int64(100)
	request := srv.Files.List().
		Context(ctx).
		PageSize(pageSize).
		Corpora("user").
		Q("'" + rootId + "' in parents and mimeType='image/jpeg'").
		Fields("nextPageToken,files(id, name, parents)")

	ch := make(chan *drive.File, pageSize)
	wg := sync.WaitGroup{}

	for x := 0; x < 3; x++ {
		wg.Add(1)
		y := x
		go func() {
			for f := range ch {
				mf := drive.File{}
				_, err := srv.Files.Update(f.Id, &mf).Context(ctx).RemoveParents(rootId).AddParents(rootFilesId).Fields("").Do()
				if err != nil {
					log.Fatalf("Failed to move file: %v", err)
				}
				fmt.Printf("  --%d - file moved successfully: %s (%s)\n", y, f.Name, f.Id)
			}
			wg.Done()
		}()
	}

	for page := 0; ; page++ {
		r, err := request.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
		}

		for _, f := range r.Files {
			fmt.Printf("page %d  %s (%s)\n", page, f.Name, f.Id)
			ch <- f
		}

		if r.NextPageToken == "" {
			break
		}

		request.PageToken(r.NextPageToken)
	}

	close(ch)
	wg.Wait()
}

func deleteFiles(ctx context.Context, srv *drive.Service) {
	rootFilesId := GetRootFilesId(srv)

	var percent = 1

	fmt.Println("Files:")

	pageSize := int64(100)
	request := srv.Files.List().
		Context(ctx).
		PageSize(pageSize).
		Corpora("user").
		Q("'" + rootFilesId + "' in parents").
		Fields("nextPageToken,files(id, name)")

	ch := make(chan *drive.File, pageSize)
	wg := sync.WaitGroup{}

	for x := 0; x < 3; x++ {
		wg.Add(1)
		y := x
		go func() {
			for f := range ch {
				if !ShouldDeleteFile(f.Name, percent) {
					continue
				}
				err := srv.Files.Delete(f.Id).Do()
				if err != nil {
					log.Fatalf("Failed to delete file: %v", err)
				}
				fmt.Printf("  --%d - file deleted successfully: %s (%s)\n", y, f.Name, f.Id)
			}
			wg.Done()
		}()
	}

	for page := 0; ; page++ {
		r, err := request.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
		}

		for _, f := range r.Files {
			ch <- f
		}

		fmt.Printf("Processed page %d\n", page)

		if r.NextPageToken == "" {
			break
		}

		request.PageToken(r.NextPageToken)
	}

	close(ch)
	wg.Wait()
}
