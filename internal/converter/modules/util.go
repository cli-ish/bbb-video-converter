package modules

import (
	"encoding/json"
	"errors"
	"github.com/cli-ish/bbb-video-converter/internal/config"
	"log"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

type Video struct {
	VideoPath   string
	Duration    float64
	Width       float64
	Height      float64
	IsOnlyAudio bool
}

type ParseInfo struct {
	Streams []ParseInfoStream `json:"streams"`
}

type ParseInfoStream struct {
	Width    int    `json:"width"`
	Heigth   int    `json:"height"`
	Duration string `json:"duration"`
}

func GetVideoInfo(videofile string) (Video, error) {
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height,duration", "-of", "json", videofile).Output()
	if err != nil {
		return Video{}, err
	}
	var info ParseInfo
	err = json.Unmarshal(out, &info)
	if err != nil {
		log.Fatal(err)
	}
	if len(info.Streams) == 0 {
		return Video{}, errors.New("webcam video does not have any streams")
	}
	duration, _ := strconv.ParseFloat(info.Streams[0].Duration, 64)
	return Video{
		VideoPath:   videofile,
		Duration:    duration,
		Width:       float64(info.Streams[0].Width),
		Height:      float64(info.Streams[0].Heigth),
		IsOnlyAudio: false,
	}, nil
}

func (v *Video) IsAllWhiteVideo(duration int, config config.Data) bool {
	out, err := exec.Command("ffmpeg", "-i", v.VideoPath, "-threads", config.ThreadCount, "-vf", "negate,blackdetect=d=2:pix_th=0.00", "-an", "-f", "null", "-").CombinedOutput()
	if err != nil {
		return false
	}
	result := string(out)
	idxFind := strings.Index(result, "blackdetect")
	if idxFind == -1 {
		return false
	}
	right := strings.Index(result[idxFind:], "\n")
	blackdetect := result[idxFind : idxFind+right]
	needle := "black_duration:"
	idxFind = strings.Index(blackdetect, needle)
	if idxFind == -1 && idxFind+len(needle) < len(blackdetect) {
		return false
	}
	durationBlack, err := strconv.ParseFloat(blackdetect[idxFind+len(needle):], 64)
	if err != nil {
		return false
	}
	return math.Abs(float64(duration)-durationBlack) < 1.0
}
