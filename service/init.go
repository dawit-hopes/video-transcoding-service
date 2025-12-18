package service

type Service struct {
	ProgressUI ProgressUIService
	Transcode  TranscodeService
}

func InitService() *Service {
	progressUI := NewProgressUI()
	return &Service{
		ProgressUI: progressUI,
		Transcode:  NewTranscodeService(progressUI),
	}
}
