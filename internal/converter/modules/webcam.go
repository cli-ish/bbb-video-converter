package modules

import (
	"errors"
	"github.com/cli-ish/bbb-video-converter/internal/config"
	"os"
	"path"
)

func GetWebcamVideos(config config.Data, duration int) (Video, error) {
	webcamPath := ""
	formats := []string{"mp4", "webm"}
	for _, format := range formats {
		webcamPathTmp := path.Join(config.RecordingDir, "video", "webcams."+format)
		_, err := os.Stat(webcamPathTmp)
		if err == nil {
			webcamPath = webcamPathTmp
			break
		}
	}
	if webcamPath == "" {
		return Video{}, errors.New("no webcam video found (allowed formats mp4 and webm)")
	}
	videoInfo, err := GetVideoInfo(webcamPath)
	if err != nil {
		return Video{}, err
	}
	videoInfo.IsOnlyAudio = videoInfo.IsAllWhiteVideo(duration, config)
	return videoInfo, nil
}
