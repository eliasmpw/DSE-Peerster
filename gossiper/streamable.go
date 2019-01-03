package gossiper

import (
	"fmt"
	"github.com/eliasmpw/Peerster/common"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func transcodeStreamableFile(fileName string) {
	absPath, err := filepath.Abs("")
	common.CheckError(err)
	path := absPath +
		string(os.PathSeparator) +
		myGossiper.sharedFilesDir +
		fileName
	nameWithoutExtension := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	outputPath := absPath +
		string(os.PathSeparator) +
		myGossiper.sharedFilesDir +
		"streaming" +
		string(os.PathSeparator) +
		nameWithoutExtension +
		".mp3"
	// Audio Streaming - try to transcode to mp3 file if it is an audio file
	cmd := "ffmpeg"

	args := []string{"-y", "-i", path, "-error-resilient", "1", "-codec:a", "libmp3lame", "-b:a", "128k", outputPath}
	//args := []string{ "-y", "-i", path, "-error-resilient", "1", "-codec:a", "libmp3lame", "-qscale:a", "5", "-f", "mp3", outputPath }

	//args := []string{ "-y", "-i", path, "-f", "mp3", outputPath }

	//args := []string{ "-y", "-i", path, "-f", "wav", outputPath }

	_, err = exec.Command(cmd, args...).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "There was an error on encoding command: ", err)
		os.Exit(1)
	}

	fmt.Println("ENCODING SUCCEED")
}
