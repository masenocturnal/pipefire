package directdebit

//ArchiveConfig configuration for the archive task
type ArchiveConfig struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}
