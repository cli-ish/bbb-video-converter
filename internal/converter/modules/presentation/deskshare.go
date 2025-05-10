package presentation

import (
	"encoding/xml"
	"github.com/cli-ish/bbb-video-converter/internal/config"
	"github.com/cli-ish/bbb-video-converter/internal/converter/modules"
	"io"
	"os"
	"path"
)

type deskshareData struct {
	VideoParts []videoPart
	Video      modules.Video
}

type videoPart struct {
	Start  float64
	End    float64
	Width  float64
	Height float64
}

type recordingDeskshare struct {
	XMLName xml.Name         `xml:"recording"`
	Events  []eventDeskshare `xml:"event"`
}

type eventDeskshare struct {
	Start  float64 `xml:"start_timestamp,attr"`
	End    float64 `xml:"stop_timestamp,attr"`
	Width  float64 `xml:"video_width,attr"`
	Height float64 `xml:"video_height,attr"`
}

func parseDeskshares(config config.Data) deskshareData {
	desksharePath := path.Join(config.RecordingDir, "deskshare.xml")
	_, err := os.Stat(desksharePath)
	if !os.IsNotExist(err) {
		deskshareFile, err := os.Open(desksharePath)
		if err == nil {
			defer deskshareFile.Close()
			byteValue, _ := io.ReadAll(deskshareFile)
			var rec recordingDeskshare
			err = xml.Unmarshal(byteValue, &rec)
			data := deskshareData{}
			for _, v := range rec.Events {
				data.VideoParts = append(data.VideoParts, videoPart{v.Start, v.End, v.Width, v.Height})
			}
			data.Video = modules.Video{}
			formats := []string{"mp4", "webm"}
			for _, format := range formats {
				webcamPathTmp := path.Join(config.RecordingDir, "deskshare/deskshare."+format)
				_, err := os.Stat(webcamPathTmp)
				if err == nil {
					data.Video.VideoPath = webcamPathTmp
					break
				}
			}
			return data
		}
	}
	return deskshareData{}
}
