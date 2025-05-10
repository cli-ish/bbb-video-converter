package modules

import (
	"errors"
	"fmt"
	"github.com/cli-ish/bbb-video-converter/internal/config"
	"github.com/cli-ish/bbb-video-converter/internal/util"
	"math"
	"os"
	"path"
)

func CombinePresentationWithWebcams(presentation Video, webcam Video, config config.Data) (Video, error) {
	videoPath := path.Join(config.WorkingDir, "out.mp4")
	if presentation.VideoPath == "" && webcam.VideoPath == "" {
		return Video{}, errors.New("the presentation does not contain any renderable inputs (slides, deskshares or webcams/audio)")
	}
	if presentation.VideoPath != "" && webcam.VideoPath == "" {
		err := os.Rename(presentation.VideoPath, videoPath)
		if err != nil {
			return Video{}, errors.New("could not rename presentation video")
		}
	}
	if presentation.VideoPath == "" && webcam.VideoPath != "" {
		err := copyWebcamsVideo(webcam, videoPath, config)
		if err != nil {
			return Video{}, errors.New("webcam video copy crashed")
		}
	}
	// webcam is only audio not laoded ?
	if presentation.VideoPath != "" && webcam.IsOnlyAudio {
		err := copyWebcamsAudioToPresentation(presentation, webcam, videoPath, config)
		if err != nil {
			return Video{}, errors.New("copy webcam audio crashed")
		}
	} else {
		err := stackWebcamsToPresentation(presentation, webcam, videoPath, config)
		if err != nil {
			return Video{}, errors.New("could not stack webcam and presentation")
		}
	}
	return GetVideoInfo(videoPath)
}

func copyWebcamsVideo(webcam Video, videoPath string, config config.Data) error {
	_, err := util.ExecuteCommand("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", webcam.VideoPath, "-y", videoPath).Output()
	if err != nil {
		return err
	}
	return nil
}

func copyWebcamsAudioToPresentation(presentation Video, webcam Video, videoPath string, config config.Data) error {
	_, err := util.ExecuteCommand("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", presentation.VideoPath, "-i", webcam.VideoPath, "-c:v", "copy", "-c:a", "aac", "-map", "0:0", "-map", "1:1", "-shortest", "-preset", "ultrafast", "-y", videoPath).Output()
	if err != nil {
		return err
	}
	return nil
}

func stackWebcamsToPresentation(presentation Video, webcam Video, videoPath string, config config.Data) error {
	width := presentation.Width + webcam.Width
	height := math.Max(presentation.Height, webcam.Height)
	if int(height)%2 == 1 {
		height += 1
	}
	_, err := util.ExecuteCommand("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", presentation.VideoPath, "-i", webcam.VideoPath, "-filter_complex", "[0:v]pad=width="+fmt.Sprint(width)+":height="+fmt.Sprint(height)+":color=white[p];[p][1:v]overlay=x="+fmt.Sprint(presentation.Width)+":y=0[out]", "-map", "[out]", "-map", "1:1", "-c:a", "aac", "-shortest", "-y", videoPath).Output()
	if err != nil {
		return err
	}
	return nil
}

func ProcessToEndExtension(input Video, config config.Data) error {
	_, err := util.ExecuteCommand("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", input.VideoPath, "-y", config.OutputFile).Output()
	if err != nil {
		return err
	}
	return nil
}
