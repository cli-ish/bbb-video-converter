package presentation

import (
	"bbb-video-converter/config"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type screenSize struct {
	Width  int
	Height int
}

type FrameInfo struct {
	FilePath  string
	Timestamp float64
}

var frameInfos map[float64]FrameInfo

func captureFrames(config config.Data, presentation Presentation) (map[float64]FrameInfo, error) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoDefaultBrowserCheck,
		chromedp.NoFirstRun,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
	}
	ctx, cancelA := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelA()
	ctxReal, cancelB := chromedp.NewContext(ctx)
	defer cancelB()
	frameInfos = make(map[float64]FrameInfo)
	if err := chromedp.Run(ctxReal, renderFrames(config, presentation)); err != nil {
		return frameInfos, err
	}
	return frameInfos, nil
}

func renderFrames(config config.Data, presentation Presentation) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(`file://` + path.Join(config.RecordingDir, "/shapes.svg")),
		chromedp.ActionFunc(func(ctx context.Context) error {
			size := screenSize{0, 0}
			frames := presentation.Frames
			timestamps := make([]float64, 0, len(frames))
			for k := range frames {
				timestamps = append(timestamps, k)
			}
			sort.Float64s(timestamps)
			defineFunctions(ctx)
			var wg sync.WaitGroup
			for _, timestamp := range timestamps {
				frame := frames[timestamp]
				actionString := ""
				for _, action := range frame.Actions {
					switch action.Name {
					case ShowImage:
						actionString += showImage(action)
						size.Width = action.Width
						size.Height = action.Height
						actionString += showCursor()
						break
					case HideImage:
						actionString += hideImage(action)
						actionString += hideCursor()
						break
					case ShowDrawing:
						actionString += showDrawing(action)
						break
					case HideDrawing:
						actionString += hideDrawing(action)
						break
					case SetViewBox:
						actionString += setViewBox(action)
						break
					case MoveCursor:
						parts := strings.Split(action.Value, " ")
						posx, _ := strconv.ParseFloat(parts[0], 64)
						posy, _ := strconv.ParseFloat(parts[1], 64)
						posx = posx * float64(size.Width)
						posy = posy * float64(size.Height)
						actionString += moveCursor(posx, posy)
						break
					}
				}
				_, _, _ = runtime.Evaluate(actionString).Do(ctx)
				var buf []byte
				err := chromedp.FullScreenshot(&buf, 90).Do(ctx)
				if err != nil {
					log.Fatal(err)
					return err
				}
				wg.Add(1)
				go func(stamp float64, buffer []byte) {
					defer wg.Done()
					h := sha256.New()
					h.Write(buffer)
					sum := hex.EncodeToString(h.Sum(nil))
					iPath := path.Join(config.WorkingDir, sum+".png")
					frameInfos[stamp] = FrameInfo{iPath, stamp}
					_, err = os.Stat(iPath)
					if os.IsNotExist(err) {
						if errWrite := ioutil.WriteFile(iPath, buffer, 0o644); errWrite != nil {
							log.Fatal(errWrite)
						}
					}
				}(timestamp, buf)
			}
			wg.Wait()
			return nil
		}),
	}
}

func defineFunctions(ctx context.Context) {
	functions := []string{
		"var svgfile=document.querySelector('#svgfile');svgfile.innerHTML+='<circle id=\"cursor\" cx=\"9999\" cy=\"9999\" r=\"5\" stroke=\"red\" stroke-width=\"3\" fill=\"red\" style=\"visibility:hidden\" />';var cursor=document.querySelector('#cursor');",
		"function showImage(id){let el=document.querySelector('#'+id).style.visibility='visible';let canvas=document.querySelector('#canvas'+id.match(/\\d+/));if(canvas){canvas.setAttribute('display','block');}}",
		"function hideImage(id){let el=document.querySelector('#'+id).style.visibility='hidden';let canvas=document.querySelector('#canvas'+id.match(/\\d+/));if(canvas){canvas.setAttribute('display','none');}}",
		"function showDrawing(id){let drawing=document.querySelector('#'+id);document.querySelectorAll('[shape='+drawing.getAttribute('shape')+']').forEach(element=>{element.style.visibility='hidden'});drawing.style.visibility='visible';}",
		"function hideDrawing(id){document.querySelector('#'+id).style.display='none';}",
		"function setViewBox(viewbox){svgfile.setAttribute('viewBox',viewbox);}",
		"function moveCursor(posx, posy){cursor.setAttribute('cx',posx);cursor.setAttribute('cy',posy);}",
		"function showCursor(){cursor.style.visibility='visible';}",
		"function hideCursor(){cursor.style.visibility='hidden';}",
	}
	_, _, _ = runtime.Evaluate(strings.Join(functions, "")).Do(ctx)
}

func showImage(action Action) string {
	return "showImage('" + action.Id + "');"
}

func hideImage(action Action) string {
	return "hideImage('" + action.Id + "');"
}

func showDrawing(action Action) string {
	return "showDrawing('" + action.Id + "');"
}

func hideDrawing(action Action) string {
	return "hideDrawing('" + action.Id + "');"
}

func setViewBox(action Action) string {
	return "setViewBox('" + action.Value + "');"
}

func moveCursor(posx float64, posy float64) string {
	return "moveCursor(" + fmt.Sprint(posx) + "," + fmt.Sprint(posy) + ");"
}

func showCursor() string {
	return "showCursor();"
}

func hideCursor() string {
	return "hideCursor();"
}
