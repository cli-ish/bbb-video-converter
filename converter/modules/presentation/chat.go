package presentation

import (
	"bbb-video-converter/config"
	"bbb-video-converter/converter/modules"
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	_ "embed"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"sync"
)

//go:embed "templates/chat.gohtml"
var chatMessageTemplate embed.FS

type popcorn struct {
	XMLName  xml.Name      `xml:"popcorn"`
	Messages []chatMessage `xml:"chattimeline"`
}

type chatMessage struct {
	In     float64 `xml:"in,attr"`
	Author string  `xml:"name,attr"`
	Text   string  `xml:"message,attr"`
	Target string  `xml:"target,attr"`
}

type ChatMessageSimplified struct {
	In     float64
	Author string
	Text   string
}

func GetChatMessages(config config.Data) (map[float64][]ChatMessageSimplified, error) {
	chatMessagesPath := path.Join(config.RecordingDir, "slides_new.xml")
	_, err := os.Stat(chatMessagesPath)
	if err != nil {
		return map[float64][]ChatMessageSimplified{}, nil
	}
	xmlFile, err := os.Open(chatMessagesPath)
	defer xmlFile.Close()
	if err != nil {
		return map[float64][]ChatMessageSimplified{}, err
	}
	byteValue, _ := ioutil.ReadAll(xmlFile)
	var xmldata popcorn
	err = xml.Unmarshal(byteValue, &xmldata)
	if err != nil {
		return map[float64][]ChatMessageSimplified{}, err
	}
	returnMessages := make(map[float64][]ChatMessageSimplified)
	for _, v := range xmldata.Messages {
		if v.Target == "chat" {
			data, ok := returnMessages[v.In]
			if !ok {
				data = []ChatMessageSimplified{}
			}
			data = append(data, ChatMessageSimplified{v.In, v.Author, v.Text})
			returnMessages[v.In] = data
		}
	}
	return returnMessages, nil
}

func RenderMessageFrames(messageList map[float64][]ChatMessageSimplified, config config.Data, duration int) modules.Video {
	tmpl := template.Must(template.ParseFS(chatMessageTemplate, "templates/chat.gohtml"))
	var messages []ChatMessageSimplified
	for _, m := range messageList {
		messages = append(messages, m...)
	}
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].In < messages[j].In
	})
	var tpl bytes.Buffer
	err := tmpl.Execute(&tpl, messages)
	if err != nil {
		return modules.Video{}
	}
	options := GetChromeDpSettings()
	options = append(options, chromedp.WindowSize(192, 600))
	browserCtx, cancelA := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancelA()
	frameInfos := make(map[float64]FrameInfo)
	ctx, cancelMe := chromedp.NewContext(browserCtx)
	defer cancelMe()
	if err = chromedp.Run(ctx); err != nil {
		log.Println("start:", err)
	}
	if err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, tpl.String()).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			defineChatFunctions(ctx)
			var wg sync.WaitGroup
			saveScreen := func(stamp float64, buffer []byte) {
				defer wg.Done()
				h := sha256.New()
				h.Write(buffer)
				sum := hex.EncodeToString(h.Sum(nil))
				iPath := path.Join(config.WorkingDir, sum+".png")
				frameInfos[stamp] = FrameInfo{FilePath: iPath, Timestamp: stamp}
				_, err = os.Stat(iPath)
				if os.IsNotExist(err) {
					if errWrite := ioutil.WriteFile(iPath, buffer, 0o644); errWrite != nil {
						log.Fatal(errWrite)
					}
				}
			}
			_, ok := messageList[0]
			if !ok {
				var buf []byte
				err = chromedp.FullScreenshot(&buf, 90).Do(ctx)
				if err != nil {
					log.Fatal(err)
					return err
				}
				wg.Add(1)
				go saveScreen(0, buf)
			}
			for timestamp, _ := range messageList {
				if timestamp < float64(duration) {
					actionString := "s(" + fmt.Sprint(timestamp) + ");"
					_, _, _ = runtime.Evaluate(actionString).Do(ctx)
					var buf []byte
					err = chromedp.FullScreenshot(&buf, 90).Do(ctx)
					if err != nil {
						log.Fatal(err)
						return err
					}
					wg.Add(1)
					go saveScreen(timestamp, buf)
				}
			}
			wg.Wait()
			return nil
		}),
	}); err != nil {
		log.Fatal(err)
	}
	return renderChatVideo(config, frameInfos, duration)
}

func defineChatFunctions(ctx context.Context) {
	functions := []string{
		"function s(inp){let el=document.querySelector('.container');el.scrollTop=el.scrollTopMax;el.querySelectorAll(\"div[data-time='\"+inp+\"']\").forEach(x => {x.classList.add('shown');});}",
	}
	_, _, _ = runtime.Evaluate(strings.Join(functions, "")).Do(ctx)
}


func renderChatVideo(config config.Data, infos map[float64]FrameInfo, durationReal int) modules.Video {
	timestamps := make([]float64, 0, len(infos))
	for k := range infos {
		timestamps = append(timestamps, k)
	}
	sort.Float64s(timestamps)
	slidesContent := ""
	for i, timestamp := range timestamps {
		slidesContent += "file '" + infos[timestamp].FilePath + "'\n"
		if i+1 != len(timestamps) {
			duration := float64(math.Round(10*(timestamps[i+1]-timestamp)) / 10)
			slidesContent += "duration " + fmt.Sprintf("%f", duration) + "\n"
		}
	}
	slidesContent += "duration " + fmt.Sprint(float64(durationReal)-timestamps[len(timestamps)-1]) + "\n"
	slidesContent += "file '" + infos[timestamps[len(timestamps)-1]].FilePath + "'\n"
	slidesTxtFile := path.Join(config.WorkingDir, "chat.txt")
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
	result.VideoPath = path.Join(config.WorkingDir, "chat.mp4")
	_, err = exec.Command("ffmpeg", "-safe", "0", "-hide_banner", "-loglevel", "error", "-f", "concat", "-i", slidesTxtFile, "-threads", config.ThreadCount, "-y", "-strict", "-2", "-crf", "22", "-preset", "ultrafast", "-c", "copy", "-pix_fmt", "yuv420p", result.VideoPath).Output()
	if err != nil {
		return modules.Video{}
	}
	result, err = modules.GetVideoInfo(result.VideoPath)
	if err != nil {
		return modules.Video{}
	}
	return result
}
