package download

// DownloadResult holds the result of a download operation.
type DownloadResult struct {
	OutputPath string
	UploadDate string // YYYYMMDD format or empty
	TempDir    string
}
