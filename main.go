package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func getYtInitialData(url string) (interface{}, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	html, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	startIndex := strings.Index(string(html), "var ytInitialData = {")
	if startIndex == -1 {
		return "", errors.New("could not find var ytInitialData = {")
	}

	startIndex += len("var ytInitialData = ")

	endIndex := strings.Index(string(html[startIndex:]), "</script>")
	if endIndex == -1 {
		return "", errors.New("could not find </script>")
	}

	endIndex += startIndex

	var data interface{}
	if err := json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimSpace(string(html[startIndex:endIndex])), ";")), &data); err != nil {
		return "", err
	}

	return data, nil
}

func main() {
	currentlyDownloading := make(map[string]struct{})

	for {
		data, err := getYtInitialData("https://www.youtube.com/@" + strings.TrimPrefix(strings.TrimPrefix(os.Args[1], "https://www.youtube.com/"), "@") + "/streams")
		if err != nil {
			fmt.Println(err.Error())
			time.Sleep(time.Minute)
			continue
		}

		tabs, ok := data.(map[string]interface{})["contents"].(map[string]interface{})["twoColumnBrowseResultsRenderer"].(map[string]interface{})["tabs"].([]interface{})
		if !ok {
			fmt.Println("could not find tabs")
			time.Sleep(time.Minute)
			continue
		}

		for _, tab := range tabs {
			tab := tab.(map[string]interface{})["tabRenderer"]
			if tab == nil {
				// fmt.Println("could not find tabRenderer")
				continue
			}

			if tab.(map[string]interface{})["title"] == "Live" {
				items := tab.(map[string]interface{})["content"].(map[string]interface{})["richGridRenderer"].(map[string]interface{})["contents"].([]interface{})
				for _, item := range items {
					richItemRenderer := item.(map[string]interface{})["richItemRenderer"]
					if richItemRenderer == nil {
						// fmt.Println("could not find richItemRenderer")
						continue
					}

					rendererContent := richItemRenderer.(map[string]interface{})["content"].(map[string]interface{})
					video := rendererContent["videoRenderer"].(map[string]interface{})
					videoId := video["videoId"].(string)
					thumbnailOverlays := video["thumbnailOverlays"].([]interface{})
					var isLive bool
					for _, overlay := range thumbnailOverlays {
						overlayTime, ok := overlay.(map[string]interface{})["thumbnailOverlayTimeStatusRenderer"]
						if ok && overlayTime.(map[string]interface{})["style"] == "LIVE" {
							isLive = true
							break
						}
					}

					if !isLive {
						continue
					}

					if _, ok := currentlyDownloading[videoId]; ok {
						continue
					}

					args := []string{"new-tab", "ytarchive", "https://www.youtube.com/watch?v=" + videoId}
					args = append(args, os.Args[2:]...)
					exec.Command("wt", args...).Run()
					currentlyDownloading[videoId] = struct{}{}
				}
			}
		}

		time.Sleep(time.Minute)
	}
}
