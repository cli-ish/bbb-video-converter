package presentation

import (
	"bbb-video-converter/config"
	"bbb-video-converter/converter/modules"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"sort"
	"sync"
	"time"
)

type Presentation struct {
	Frames map[float64]Frame
	Width  float64
	Height float64
}

type Frame struct {
	Timestamp float64
	Actions   []Action
}

type Action struct {
	Name   string
	Id     string
	Width  int
	Height int
	Value  string
}

const (
	ShowImage   string = "showImage"
	HideImage   string = "hideImage"
	ShowDrawing string = "showDrawing"
	SetViewBox  string = "setViewBox"
	MoveCursor  string = "moveCursor"
	HideDrawing string = "hideDrawing"
)

func renderSlides(config config.Data, duration int) modules.Video {
	presentation := parseSlidesData(config.RecordingDir, duration)
	if len(presentation.Frames) > 1 {
		start := time.Now()
		infos, err := captureFrames(config, presentation)
		if err != nil {
			return modules.Video{}
		}
		end := time.Now().Sub(start)
		log.Println("slide generation took: " + fmt.Sprint(end))
		start = time.Now()
		video := renderVideo(presentation, config, infos, duration)
		end = time.Now().Sub(start)
		log.Println("slide.mp4 creation took: " + fmt.Sprint(end))
		return video
	}
	return modules.Video{}
}

func renderVideo(presentation Presentation, config config.Data, infos map[float64]FrameInfo, durationReal int) modules.Video {
	frames := presentation.Frames
	timestamps := make([]float64, 0, len(frames))
	for k := range frames {
		timestamps = append(timestamps, k)
	}
	sort.Float64s(timestamps)
	slidesContent := ""
	for i, timestamp := range timestamps {
		slidesContent += "file '" + infos[timestamp].FilePath + "'\n"
		if i+1 != len(timestamps) {
			duration := math.Round(10*(timestamps[i+1]-timestamp)) / 10
			slidesContent += "duration " + fmt.Sprint(duration) + "\n"
		}
	}
	slidesTxtFile := path.Join(config.WorkingDir, "slides.txt")
	file, err := os.Create(slidesTxtFile)
	if err != nil {
		return modules.Video{}
	}
	defer file.Close()
	_, err = file.WriteString(slidesContent)
	if err != nil {
		return modules.Video{}
	}
	result := modules.Video{}
	result.VideoPath = path.Join(config.WorkingDir, "slides.mp4")
	_, err = exec.Command("ffmpeg", "-safe", "0", "-hide_banner", "-loglevel", "error", "-f", "concat", "-i", slidesTxtFile, "-threads", config.ThreadCount, "-y", "-strict", "-2", "-crf", "22", "-preset", "ultrafast", "-t", fmt.Sprint(durationReal), "-c", "copy", "-pix_fmt", "yuv420p", result.VideoPath).Output()
	if err != nil {
		return modules.Video{}
	}
	return result
}

func parseSlidesData(recordingDir string, duration int) Presentation {
	var frames map[float64]Frame
	var wg sync.WaitGroup
	var width float64
	var height float64
	var panFrames map[float64]Frame
	var panCursors map[float64]Frame
	wg.Add(1)
	go func() {
		defer wg.Done()
		frames, width, height = parseShapes(recordingDir, duration)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		panFrames = parsePanzooms(recordingDir, duration)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		panCursors = parseCursors(recordingDir, duration)
	}()
	wg.Wait()
	frames = mergeFrames(frames, panFrames)
	frames = mergeFrames(frames, panCursors)
	pres := Presentation{frames, width, height}
	return pres
}

func mergeFrames(old map[float64]Frame, newFrame map[float64]Frame) map[float64]Frame {
	for key, value := range newFrame {
		_, ok := old[key]
		if !ok {
			old[key] = value
			continue
		}
		frame := old[key]
		for _, action := range value.Actions {
			frame.Actions = append(old[key].Actions, action)
		}
		old[key] = frame
	}
	return old
}
