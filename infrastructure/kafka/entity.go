package kafka

type TranscodeJob struct {
	VideoID   string  `json:"video_id"`
	FilePath  string  `json:"file_path"`
	VideoName string  `json:"video_name"`
	Duration  float64 `json:"duration"`
	MaxHeight int     `json:"max_height"` // To prevent upscaling!
}
