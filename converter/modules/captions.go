package modules

import (
	"bbb-video-converter/config"
	"bbb-video-converter/converter/modules/langs"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

type caption struct {
	Locale string `json:"locale"`
}

type Caption struct {
	Code string
	File string
}

func CreateCaptions(config config.Data) ([]Caption, error) {
	captionPath := path.Join(config.RecordingDir, "captions.json")
	_, err := os.Stat(captionPath)
	if err != nil {
		return []Caption{}, nil
	}
	xmlFile, err := os.Open(captionPath)
	defer xmlFile.Close()
	if err != nil {
		return []Caption{}, err
	}
	byteValue, _ := ioutil.ReadAll(xmlFile)
	var captions []caption
	err = json.Unmarshal(byteValue, &captions)
	if err != nil {
		return []Caption{}, err
	}
	var returnCaptions []Caption
	for _, v := range captions {
		capt, err := transformCaptions(config, v.Locale)
		if err != nil {
			continue
		}
		returnCaptions = append(returnCaptions, capt)
	}
	return returnCaptions, nil
}

func AddCaption(captions []Caption, config config.Data, fullVideo Video) error {
	tmpFile := path.Join(config.WorkingDir, "video.caption.tmp.mp4")
	cmd := []string{"-hide_banner", "-loglevel", "error", "-threads", "1", "-i", fullVideo.VideoPath}
	for _, v := range captions {
		cmd = append(cmd, "-i", v.File)
	}
	cmd = append(cmd, "-map", "0")
	for i := range captions {
		cmd = append(cmd, "-map", fmt.Sprint(i+1)+":s")
	}
	cmd = append(cmd, "-c", "copy")
	for range captions {
		cmd = append(cmd, "-c:s", "mov_text")
	}
	for i, v := range captions {
		cmd = append(cmd, "-metadata:s:s:"+fmt.Sprint(i), "language="+v.Code)
	}
	cmd = append(cmd, "-y", tmpFile)
	_, err := exec.Command("ffmpeg", cmd...).Output()
	if err != nil {
		return err
	}
	err = os.Remove(fullVideo.VideoPath)
	if err != nil {
		return err
	}
	err = os.Rename(tmpFile, fullVideo.VideoPath)
	if err != nil {
		return err
	}
	return nil
}

func transformCaptions(config config.Data, Locale string) (Caption, error) {
	captionCode := langs.LanguageList[Locale].Two
	captionInFile := path.Join(config.RecordingDir, "caption_"+Locale+".vtt")
	captionOutFile := path.Join(config.WorkingDir, "caption_"+captionCode+".srt")
	_, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "-warning", "-i", captionInFile, captionOutFile).Output()
	if err != nil {
		return Caption{}, err
	}
	return Caption{captionCode, captionOutFile}, nil
}
