package presentation

import (
	"encoding/xml"
	"io"
	"os"
	"path"
)

type recording struct {
	XMLName xml.Name `xml:"recording"`
	Events  []event  `xml:"event"`
}

type event struct {
	Timestamp float64 `xml:"timestamp,attr"`
	ViewBox   string  `xml:"viewBox"`
}

func parsePanzooms(recordingDir string, duration int) map[float64]Frame {
	panzoomPath := path.Join(recordingDir, "panzooms.xml")
	_, err := os.Stat(panzoomPath)
	if !os.IsNotExist(err) {
		panzoomFile, err := os.Open(panzoomPath)
		if err == nil {
			defer panzoomFile.Close()
			byteValue, _ := io.ReadAll(panzoomFile)
			var rec recording
			err = xml.Unmarshal(byteValue, &rec)
			if err != nil {
				return map[float64]Frame{}
			}
			frames := map[float64]Frame{}
			for _, evt := range rec.Events {
				if evt.Timestamp < float64(duration) {
					_, ok := frames[evt.Timestamp]
					if !ok {
						frames[evt.Timestamp] = Frame{Timestamp: evt.Timestamp}
					}
					frame := frames[evt.Timestamp]
					frame.Actions = append(frame.Actions, Action{
						Name:  SetViewBox,
						Value: evt.ViewBox,
					})
					frames[evt.Timestamp] = frame
				}
			}
			return frames
		}
	}
	return map[float64]Frame{}
}
