package converter

import (
	"bbb-video-converter/config"
	"bbb-video-converter/converter/modules"
	"bbb-video-converter/converter/modules/presentation"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Converter struct {
}

func (c *Converter) Run(config config.Data) error {
	workingDir, err := ioutil.TempDir(os.TempDir(), "converter-*-data")
	if err != nil {
		return errors.New("could not create temporary working directory")
	}
	config.WorkingDir = workingDir
	defer func() {
		err = os.RemoveAll(config.WorkingDir)
		if err != nil {
			log.Println("Could not clear up tmp for (" + config.WorkingDir + ")!")
		}
		log.Println("Cleanup done.")
	}()

	duration, err := modules.GetDuration(config)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var webcamVideo modules.Video
	var presentationVideo modules.Video
	var captions []modules.Caption
	wg.Add(1)
	go func() {
		defer wg.Done()
		webcamVideo, _ = modules.GetWebcamVideos(config, duration)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		presentationVideo = presentation.CreatePresentationVideo(config, duration)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		captions, _ = modules.CreateCaptions(config)
	}()
	wg.Wait()
	start := time.Now()
	fullVideo, err := modules.CombinePresentationWithWebcams(presentationVideo, webcamVideo, config)
	if err != nil {
		return err
	}
	end := time.Now().Sub(start)
	log.Println("Combine presentation with webcam video took: " + fmt.Sprint(end))

	if len(captions) > 0 {
		err = modules.AddCaption(captions, config, fullVideo)
		if err != nil {
			// Todo: should we really exit here if the caption thrown an error ?
			return err
		}
		log.Println("Added caption data to video")
	}
	if strings.HasSuffix(config.OutputFile, ".webm") {
		err = modules.ProcessToEndExtension(fullVideo, config)
		if err != nil {
			return err
		}
	} else {
		err = copyFile(fullVideo.VideoPath, config.OutputFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(fromFile string, toFile string) error {
	srcFile, err := os.Open(fromFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	destFile, err := os.Create(toFile)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	return destFile.Sync()
}
