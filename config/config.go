package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Data struct {
	RecordingDir string
	OutputFile   string
	WorkingDir   string
}

func (c *Data) LoadConfig() error {
	showVersion := false
	flag.StringVar(&c.RecordingDir, "i", "",
		"Specify recording directory.")
	flag.StringVar(&c.OutputFile, "o", "",
		"Specify output file. Default is video.mp4 in the recording dir.")
	flag.BoolVar(&showVersion, "v", false,
		"Show current version.")
	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Println("bbb-video-converter -v")
		fmt.Println("bbb-video-converter -h")
		fmt.Println("bbb-video-converter \\\n" +
			"\t-i /var/bigbluebutton/published/presentation/{RECORDING-ID} \\\n" +
			"\t-o /var/bigbluebutton/published/presentation/{RECORDING-ID}/video.mp4")
	}
	flag.Parse()
	if showVersion {
		fmt.Println("Current version: v0.0.1-a")
		os.Exit(2)
	}
	if c.RecordingDir == "" {
		return errors.New("recording dir can not be empty")
	}
	_, err := os.Stat(c.RecordingDir)
	if os.IsNotExist(err) {
		return errors.New("recording dir can not be found (" + c.RecordingDir + ")")
	}
	if c.OutputFile == "" {
		c.OutputFile = filepath.Join(c.RecordingDir, "video.mp4")
	} else if !strings.HasPrefix(c.OutputFile, string(os.PathSeparator)) {
		c.OutputFile = filepath.Join(c.RecordingDir, c.OutputFile)
	}
	if !strings.HasSuffix(c.OutputFile, ".mp4") && !strings.HasSuffix(c.OutputFile, ".webm") {
		return errors.New("output file can only be an mp4 or webm (the file extension must match)")
	}
	outDir := filepath.Dir(c.OutputFile)
	dirInfo, err := os.Stat(outDir)
	if os.IsNotExist(err) {
		return errors.New("output dir can not be found (" + outDir + ")")
	}
	// https://stackoverflow.com/questions/45429210/how-do-i-check-a-files-permissions-in-linux-using-go
	perm := dirInfo.Mode().Perm()
	if perm&0b110000000 != 0b110000000 {
		return errors.New("output dir can not be written/read from (" + outDir + ")")
	}
	return nil
}
