package directdebit

type CleanUpConfig struct {
	Dir  []string `json:"dir"`
	File []string `json:"file"`
}

func (p ddPipeline) CleanDirtyFiles() []error {

}
