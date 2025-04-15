package domain

// ProgressDetails contains detailed information about the progress of an operation
type ProgressDetails struct {
	Stage           string  // The current stage of the operation
	StepDetail      string  // Detailed information about the current step
	ImporterName    string  // Name of the importer being used (if applicable)
	TotalTracks     int     // Total number of tracks to process
	Progress        float64 // Progress percentage (0-100)
	CurrentTrack    int     // Current track being processed
	ProcessedTracks int     // Number of tracks processed so far
	ErrorOccurred   bool    // Whether an error occurred during processing
}
