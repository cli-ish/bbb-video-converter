package modules

import (
	"bbb-video-converter/config"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"
)

func CombineVideos(presentation Video, webcam Video, chat Video, config config.Data) (Video, error) {
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
	// webcam is only audio not loaded ?
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
	info, err := GetVideoInfo(videoPath)
	if err == nil && chat.VideoPath != "" {
		full := path.Join(config.WorkingDir, "outwithchat.mp4")
		err = stackChatToVideo(info, chat, full, config)
		if err != nil {
			return Video{}, errors.New("could not stack chat to presentation")
		}
		info, err = GetVideoInfo(full)
	}
	return info, err
}

func copyWebcamsVideo(webcam Video, videoPath string, config config.Data) error {
	_, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", webcam.VideoPath, "-y", videoPath).Output()
	if err != nil {
		return err
	}
	return nil
}

func copyWebcamsAudioToPresentation(presentation Video, webcam Video, videoPath string, config config.Data) error {
	_, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", presentation.VideoPath, "-i", webcam.VideoPath, "-c:v", "copy", "-c:a", "aac", "-map", "0:0", "-map", "1:1", "-shortest", "-preset", "ultrafast", "-y", videoPath).Output()
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
	_, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", presentation.VideoPath, "-i", webcam.VideoPath, "-filter_complex", "[0:v]pad=width="+fmt.Sprint(width)+":height="+fmt.Sprint(height)+":color=white[p];[p][1:v]overlay=x="+fmt.Sprint(presentation.Width)+":y=0[out]", "-map", "[out]", "-map", "1:1", "-c:a", "aac", "-shortest", "-y", videoPath).Output()
	if err != nil {
		return err
	}
	return nil
}

func stackChatToVideo(mainVideo Video, chat Video, videoPath string, config config.Data) error {
	// Todo: merging is so lsow because there is a issue with frame per seconds or something.!!!!!!
	_, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", chat.VideoPath, "-i", mainVideo.VideoPath, "-filter_complex", "hstack", videoPath).CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}

func ProcessToEndExtension(input Video, config config.Data) error {
	_, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", input.VideoPath, "-y", config.OutputFile).Output()
	if err != nil {
		return err
	}
	return nil
}
