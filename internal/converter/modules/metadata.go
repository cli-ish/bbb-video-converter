package modules

import (
	"encoding/xml"
	"errors"
	"github.com/cli-ish/bbb-video-converter/internal/config"
	"io"
	"os"
	"path"
)

type Recording struct {
	XMLName  xml.Name `xml:"recording"`
	Playback Playback `xml:"playback"`
}

type Playback struct {
	XMLName  xml.Name `xml:"playback"`
	Duration int      `xml:"duration"`
}

func GetDuration(config config.Data) (int, error) {
	xmlFile, err := os.Open(path.Join(config.RecordingDir, "metadata.xml"))
	defer xmlFile.Close()
	if err != nil {
		return 0, errors.New("directory (" + config.RecordingDir + ") is not a bbb recording dir, the metadata.xml file is missing")
	}
	byteValue, _ := io.ReadAll(xmlFile)
	var recording Recording
	err = xml.Unmarshal(byteValue, &recording)
	if err != nil {
		return 0, err
	}
	return recording.Playback.Duration / 1000, nil
}
