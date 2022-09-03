package presentation

import (
	"bbb-video-converter/config"
	"bbb-video-converter/converter/modules"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"
)

func CreatePresentationVideo(config config.Data, duration int) modules.Video {
	var wg sync.WaitGroup
	var slideVideo modules.Video
	var deskData deskshareData
	wg.Add(1)
	go func() {
		defer wg.Done()
		slideVideo = renderSlides(config, duration)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		deskData = parseDeskshares(config)
	}()
	wg.Wait()
	if slideVideo.VideoPath == "" && deskData.Video.VideoPath == "" {
		return modules.Video{}
	}
	if slideVideo.VideoPath != "" && deskData.Video.VideoPath == "" {
		info, err := modules.GetVideoInfo(slideVideo.VideoPath)
		if err != nil {
			return modules.Video{}
		}
		return info
	}
	if slideVideo.VideoPath == "" && deskData.Video.VideoPath != "" {
		info, err := modules.GetVideoInfo(deskData.Video.VideoPath)
		if err != nil {
			return modules.Video{}
		}
		return info
	}
	start := time.Now()
	video := combinedSlidesAndDeskshares(slideVideo, deskData, config)
	end := time.Now().Sub(start)
	log.Println("presentation.mp4 merging took: " + fmt.Sprint(end))
	return video
}

func combinedSlidesAndDeskshares(slideVideo modules.Video, deskData deskshareData, config config.Data) modules.Video {
	info, err := modules.GetVideoInfo(slideVideo.VideoPath)
	if err != nil {
		return modules.Video{}
	}
	resizedDeskshareVideo := path.Join(config.WorkingDir, "deskshare.mp4")
	presentationOut := path.Join(config.WorkingDir, "presentation.mp4")
	presentationTmp := path.Join(config.WorkingDir, "presentation.tmp.mp4")
	_, err = exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", deskData.Video.VideoPath, "-vf", "scale=w="+fmt.Sprint(info.Width)+":h="+fmt.Sprint(info.Height)+":force_original_aspect_ratio=1,pad="+fmt.Sprint(info.Width)+":"+fmt.Sprint(info.Height)+":(ow-iw)/2:(oh-ih)/2:color=white", "-c:v", "libx264", "-preset", "ultrafast", resizedDeskshareVideo).Output()
	if err != nil {
		return modules.Video{}
	}
	for i, v := range deskData.VideoParts {
		presIn := slideVideo.VideoPath
		if i != 0 {
			presIn = presentationOut
		}
		_, err = exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-threads", config.ThreadCount, "-i", presIn, "-i", resizedDeskshareVideo, "-filter_complex", "[0][1]overlay=x=0:y=0:enable='between(t,"+fmt.Sprint(v.Start)+","+fmt.Sprint(v.End)+")'[out]", "-map", "[out]", "-c:a", "copy", "-c:v", "libx264", "-preset", "ultrafast", presentationTmp).Output()
		if err != nil {
			return modules.Video{}
		}
		_, err = os.Stat(presentationOut)
		if !os.IsNotExist(err) {
			err = os.Remove(presentationOut)
			if err != nil {
				log.Fatal(err)
			}
		}
		err = os.Rename(presentationTmp, presentationOut)
		if err != nil {
			log.Fatal(err)
		}
	}
	video, err := modules.GetVideoInfo(presentationOut)
	return video
}
