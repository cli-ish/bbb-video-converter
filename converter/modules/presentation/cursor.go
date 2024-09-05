package presentation

import (
	"encoding/xml"
	"io"
	"os"
	"path"
)

type cursorRecording struct {
	XMLName xml.Name      `xml:"recording"`
	Events  []cursorEvent `xml:"event"`
}

type cursorEvent struct {
	Timestamp float64 `xml:"timestamp,attr"`
	Cursor    string  `xml:"cursor"`
}

func parseCursors(recordingDir string, duration int) map[float64]Frame {
	cursorPath := path.Join(recordingDir, "cursor.xml")
	_, err := os.Stat(cursorPath)
	if !os.IsNotExist(err) {
		cursorFile, err := os.Open(cursorPath)
		if err == nil {
			defer cursorFile.Close()
			byteValue, _ := io.ReadAll(cursorFile)
			var rec cursorRecording
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
						Name:  MoveCursor,
						Value: evt.Cursor,
					})
					frames[evt.Timestamp] = frame
				}
			}
			return frames
		}
	}
	return map[float64]Frame{}
}
