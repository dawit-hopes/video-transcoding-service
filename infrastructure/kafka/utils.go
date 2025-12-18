package kafka

import (
	"fmt"
	"log"
)

var Resolutions = map[string]int{
	"360p":  360,
	"720p":  720,
	"1080p": 1080,
	"1440p": 1440, // 2K / QHD
	"2160p": 2160, // 4K / UHD
}

func filterResolutions(originalHeight int) map[string]int {
	fmt.Println(originalHeight, "original height")
	filteredResolutions := make(map[string]int)
	for folder, height := range Resolutions {
		if height <= originalHeight {
			filteredResolutions[folder] = height
		} else {
			log.Printf("Skipping %s (%dp) as it exceeds original height (%dp)", folder, height, originalHeight)
		}
	}

	if len(filteredResolutions) == 0 {
		filteredResolutions["original"] = originalHeight
	}

	fmt.Println(len(filteredResolutions))
	return filteredResolutions
}
