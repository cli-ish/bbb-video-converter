package presentation

import (
	"encoding/xml"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
)

type shapePresentation struct {
	Width  float64
	Height float64
	Frames map[float64]Frame
}

type shapes struct {
	XMLName  xml.Name  `xml:"svg"`
	Images   []image   `xml:"image"`
	Drawings []drawing `xml:"g>g"`
	ViewBox  string    `xml:"viewBox,attr"`
}
type image struct {
	XMLName xml.Name `xml:"image"`
	In      float64  `xml:"in,attr"`
	Out     float64  `xml:"out,attr"`
	Id      string   `xml:"id,attr"`
	Width   int      `xml:"width,attr"`
	Height  int      `xml:"height,attr"`
}
type drawing struct {
	XMLName   xml.Name `xml:"g"`
	Timestamp float64  `xml:"timestamp,attr"`
	Undo      float64  `xml:"undo,attr"`
	Id        string   `xml:"id,attr"`
}

func parseShapes(recordingDir string, duration int) (map[float64]Frame, float64, float64) {
	shapePath := path.Join(recordingDir, "shapes.svg")
	_, err := os.Stat(shapePath)
	if !os.IsNotExist(err) {
		shapeFile, err := os.Open(shapePath)
		if err == nil {
			defer shapeFile.Close()
			byteValue, _ := ioutil.ReadAll(shapeFile)
			var shapes shapes
			err = xml.Unmarshal(byteValue, &shapes)
			if err != nil {
				return map[float64]Frame{}, 0, 0
			}
			sp := shapePresentation{Frames: map[float64]Frame{}}
			sp.parseInitialViewBox(shapes.ViewBox)
			sp.parseImages(shapes.Images, duration)
			sp.parseDrawings(shapes.Drawings, duration)
			return sp.Frames, sp.Width, sp.Height
		}
	}
	return map[float64]Frame{}, 0, 0
}

func (sp *shapePresentation) parseInitialViewBox(viewBox string) {
	parts := strings.Split(viewBox, " ")
	if len(parts) < 3 {
		sp.Width = 0
		sp.Height = 0
		return
	}
	sp.Width, _ = strconv.ParseFloat(parts[2], 64)
	sp.Height, _ = strconv.ParseFloat(parts[3], 64)
}

func (sp *shapePresentation) parseImages(images []image, duration int) {
	for _, image := range images {
		if image.In < float64(duration) {
			frame := sp.getFrameByTimestamp(image.In)
			frame.Actions = append(frame.Actions, Action{
				Name:   ShowImage,
				Id:     image.Id,
				Width:  image.Width,
				Height: image.Height,
			})
			sp.Frames[image.In] = frame

			eventEnd := math.Max(image.Out, float64(duration))
			frame = sp.getFrameByTimestamp(eventEnd)
			frame.Actions = append(frame.Actions, Action{
				Name: HideImage,
				Id:   image.Id,
			})
			sp.Frames[eventEnd] = frame
		}
	}
}

func (sp *shapePresentation) parseDrawings(drawings []drawing, duration int) {
	for _, drawing := range drawings {
		if drawing.Timestamp < float64(duration) {
			frame := sp.getFrameByTimestamp(drawing.Timestamp)
			frame.Actions = append(frame.Actions, Action{
				Name: ShowDrawing,
				Id:   drawing.Id,
			})
			sp.Frames[drawing.Timestamp] = frame
			if drawing.Undo > 0 {
				frame = sp.getFrameByTimestamp(drawing.Undo)
				frame.Actions = append(frame.Actions, Action{
					Name: HideDrawing,
					Id:   drawing.Id,
				})
				sp.Frames[drawing.Undo] = frame
			}
		}
	}
}

func (sp *shapePresentation) getFrameByTimestamp(timestamp float64) Frame {
	_, ok := sp.Frames[timestamp]
	if !ok {
		sp.Frames[timestamp] = Frame{Timestamp: timestamp}
	}
	return sp.Frames[timestamp]
}
