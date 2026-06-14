package download

type Request struct {
	URL            string
	DownloadPath   string
	DownloaderKey  string
	SubscriptionID uint
	ReleaseID      uint
}

type TaskRef struct {
	ID string
}

type AddURLArgs struct {
	URL        string
	DstDirPath string
	Tool       string
}
