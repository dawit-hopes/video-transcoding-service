package transcoder

import (
	"fmt"
	"os"
	"path/filepath"
)

func GenerateMasterPlaylist(videoName string, results chan VariantInfo) error {
	masterPath := filepath.Join("output", videoName, "master.m3u8")
	f, err := os.Create(masterPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		return err
	}

	resultsSlice := make([]VariantInfo, 0)

	for v := range results {
		resultsSlice = append(resultsSlice, v)
	}

	resultsSlice = sortVariantsByHeight(resultsSlice)

	if len(resultsSlice) == 0 {
		return fmt.Errorf("no variant info available to generate master playlist")
	}

	for _, variant := range resultsSlice {
		line1 := fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n",
			variant.Bandwidth, variant.Width, variant.Height)
		line2 := fmt.Sprintf("%s/index.m3u8\n", variant.FolderName)

		if _, err := f.WriteString(line1); err != nil {
			return err
		}
		if _, err := f.WriteString(line2); err != nil {
			return err
		}
	}
	return nil
}

func sortVariantsByHeight(variants []VariantInfo) []VariantInfo {
	sorted := make([]VariantInfo, len(variants))
	copy(sorted, variants)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Height > sorted[j+1].Height {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	return sorted
}
