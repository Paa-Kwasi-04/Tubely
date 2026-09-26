package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type videoDimensions struct {
	Streams []videoStream `json:"streams"`
}

type videoStream struct {
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

func getVideoAspectRatio(filePath string) (string, error) {

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return "", err
	}

	var result videoDimensions
	err := json.Unmarshal(cmd.Stdout.(*bytes.Buffer).Bytes(), &result)
	if err != nil {
		return "", err
	}

	for _, stream := range result.Streams {
		if stream.Width > 0 && stream.Height > 0 {
			return getAspectRatio(stream.Width, stream.Height), nil
		}
	}
	return "", fmt.Errorf("no video stream found")
}

func getAspectRatio(width, height int64) string {
	if width == 16*height/9 {
		return "16:9"
	} else if height == 16*width/9 {
		return "9:16"
	}
	return "other"
}

func processVideoForFastStart(filePath string) (string, error){
	outputFilePath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputFilePath, nil
}


