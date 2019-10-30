package directdebit

import (
	"testing"

	"github.com/masenocturnal/pipefire/internal/crypto"
	log "github.com/sirupsen/logrus"
)

func TestGPGCLIEncryptFiles(t *testing.T) {
	providerConfig := &crypto.ProviderConfig{
		FingerPrint:        "7CA6C1593F28ADA95F657A4F984DB12818FC3718",
		SigningFingerPrint: "F1A1E55C231DCA4924B4976DB4A093826383B673",
	}
	providers := make(map[string]crypto.ProviderConfig, 1)
	providers["bnz"] = *providerConfig

	config := &EncryptFilesConfig{
		SrcDir:    "/tmp/ddrun/Pickup/",
		OutputDir: "/tmp/ddrun/Encrypted2/",
		Providers: providers,
		Enabled:   true,
	}
	logEntry := log.WithField("test", "test")
	pipeline := New("030c0eb8-883d-41c4-a220-57ee1ad49b11", logEntry)

	err := pipeline.encryptFiles(*config)
	if err != nil {
		t.Error(err)
	}
}
