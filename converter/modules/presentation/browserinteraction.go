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

const (
	CursorViewNone    int = 0
	CursorViewVisible int = 1
	CursorViewHidden  int = -1
)

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
	browserCtx, cancelA := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelA()
	frameInfos, err := renderFrames(browserCtx, config, presentation)
	return frameInfos, err
}

func renderFrames(browserCtx context.Context, config config.Data, presentation Presentation) (map[float64]FrameInfo, error) {
	frameInfos := make(map[float64]FrameInfo)
	frames := presentation.Frames
	timestamps := make([]float64, 0, len(frames))
	for k := range frames {
		timestamps = append(timestamps, k)
	}
	sort.Float64s(timestamps)
	frameCaptureThreads, err := strconv.Atoi(config.ThreadCount)
	if err != nil {
		frameCaptureThreads = 1
	}
	stepSize := len(timestamps) / frameCaptureThreads
	var coWaiter sync.WaitGroup
	var mutex = &sync.Mutex{}
	for i := 1; i < frameCaptureThreads+1; i++ {
		slotEnd := stepSize * i
		if slotEnd > len(timestamps) {
			slotEnd = len(timestamps)
		}
		var slot []float64
		if i == frameCaptureThreads {
			slot = timestamps
		} else {
			slot = timestamps[:slotEnd]
		}
		endX := slotEnd - stepSize
		if endX >= len(slot) {
			endX = len(slot)
		}
		if endX <= 0 {
			endX = 0
		}
		captureFrom := slot[endX]
		coWaiter.Add(1)
		go func(timestamps []float64, captureFrom float64) {
			defer coWaiter.Done()
			ctx, cancelMe := chromedp.NewContext(browserCtx)
			defer cancelMe()
			if err := chromedp.Run(ctx); err != nil {
				log.Println("start:", err)
			}
			if err := chromedp.Run(ctx, chromedp.Tasks{
				chromedp.Navigate("file://" + path.Join(config.RecordingDir, "/shapes.svg")),
				chromedp.ActionFunc(func(ctx context.Context) error {
					defineFunctions(ctx)
					var wg sync.WaitGroup
					actionString := ""
					size := screenSize{0, 0}
					CursorPos := ""
					viewBox := ""
					cursorView := CursorViewNone
					actionMap := make(map[string]Action)
					for _, timestamp := range timestamps {
						frame := frames[timestamp]
						for _, action := range frame.Actions {
							switch action.Name {
							case ShowImage:
								actionMap[action.Id] = action
								// Size cant be moved because it would not be set after parallel working and there for would not have a size.
								size.Width = action.Width
								size.Height = action.Height
								break
							case HideImage:
							case ShowDrawing:
							case HideDrawing:
								actionMap[action.Id] = action
								break
							case SetViewBox:
								viewBox = action.Value
								break
							case MoveCursor:
								CursorPos = action.Value
								break
							}
						}
						if captureFrom <= timestamp {
							for _, action := range actionMap {
								switch action.Name {
								case ShowImage:
									actionString += showImage(action)
									cursorView = CursorViewVisible
									break
								case HideImage:
									actionString += hideImage(action)
									cursorView = CursorViewHidden
									break
								case ShowDrawing:
									actionString += showDrawing(action)
									break
								case HideDrawing:
									actionString += hideDrawing(action)
									break
								}
							}
							actionMap = make(map[string]Action)
							if cursorView != CursorViewNone {
								if cursorView == CursorViewHidden {
									actionString += hideCursor()
								} else if cursorView == CursorViewVisible {
									actionString += showCursor()
								}
								cursorView = CursorViewNone
							}

							if CursorPos != "" {
								parts := strings.Split(CursorPos, " ")
								posx, _ := strconv.ParseFloat(parts[0], 64)
								posy, _ := strconv.ParseFloat(parts[1], 64)
								posx = posx * float64(size.Width)
								posy = posy * float64(size.Height)
								actionString += moveCursor(posx, posy)
								CursorPos = ""
							}
							if viewBox != "" {
								actionString += setViewBox(viewBox)
								viewBox = ""
							}
							_, _, _ = runtime.Evaluate(actionString).Do(ctx)
							actionString = ""
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
								mutex.Lock()
								frameInfos[stamp] = FrameInfo{iPath, stamp}
								mutex.Unlock()
								_, err = os.Stat(iPath)
								if os.IsNotExist(err) {
									if errWrite := ioutil.WriteFile(iPath, buffer, 0o644); errWrite != nil {
										log.Fatal(errWrite)
									}
								}
							}(timestamp, buf)
						}
					}
					wg.Wait()
					return nil
				}),
			}); err != nil {
				log.Fatal(err)
			}
		}(slot, captureFrom)
	}
	coWaiter.Wait()
	return frameInfos, nil
}

func defineFunctions(ctx context.Context) {
	functions := []string{
		"var svgfile=document.querySelector('#svgfile');svgfile.innerHTML+='<circle id=\"cursor\" cx=\"9999\" cy=\"9999\" r=\"5\" stroke=\"red\" stroke-width=\"3\" fill=\"red\" style=\"visibility:hidden\" />';var cursor=document.querySelector('#cursor');",
		"function sI(id){let el=document.querySelector('#'+id).style.visibility='visible';let canvas=document.querySelector('#canvas'+id.match(/\\d+/));if(canvas){canvas.setAttribute('display','block');}}",
		"function hI(id){let el=document.querySelector('#'+id).style.visibility='hidden';let canvas=document.querySelector('#canvas'+id.match(/\\d+/));if(canvas){canvas.setAttribute('display','none');}}",
		"function sD(id){let drawing=document.querySelector('#'+id);document.querySelectorAll('[shape='+drawing.getAttribute('shape')+']').forEach(element=>{element.style.visibility='hidden'});drawing.style.visibility='visible';}",
		"function hD(id){document.querySelector('#'+id).style.display='none';}",
		"function sVB(viewbox){svgfile.setAttribute('viewBox',viewbox);}",
		"function mC(posx, posy){cursor.setAttribute('cx',posx);cursor.setAttribute('cy',posy);}",
		"function sC(){cursor.style.visibility='visible';}",
		"function hC(){cursor.style.visibility='hidden';}",
	}
	_, _, _ = runtime.Evaluate(strings.Join(functions, "")).Do(ctx)
}

func showImage(action Action) string {
	return "sI('" + action.Id + "');"
}

func hideImage(action Action) string {
	return "hI('" + action.Id + "');"
}

func showDrawing(action Action) string {
	return "sD('" + action.Id + "');"
}

func hideDrawing(action Action) string {
	return "hD('" + action.Id + "');"
}

func setViewBox(viewBox string) string {
	return "sVB('" + viewBox + "');"
}

func moveCursor(posx float64, posy float64) string {
	return "mC(" + fmt.Sprint(posx) + "," + fmt.Sprint(posy) + ");"
}

func showCursor() string {
	return "sC();"
}

func hideCursor() string {
	return "hC();"
}
