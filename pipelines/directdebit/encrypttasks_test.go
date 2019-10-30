package directdebit

import (
	"testing"

	"github.com/masenocturnal/pipefire/internal/crypto"
	log "github.com/sirupsen/logrus"
)

// func TestGPGCLIEncryptFiles(t *testing.T) {
// 	providerConfig := &crypto.ProviderConfig{
// 		FingerPrint:        "7CA6C1593F28ADA95F657A4F984DB12818FC3718",
// 		SigningFingerPrint: "F1A1E55C231DCA4924B4976DB4A093826383B673",
// 	}
// 	providers := make(map[string]crypto.ProviderConfig, 1)
// 	providers["bnz"] = *providerConfig

// 	config := &EncryptFilesConfig{
// 		SrcDir:    "/tmp/ddrun/Pickup/",
// 		OutputDir: "/tmp/ddrun/Encrypted2/",
// 		Providers: providers,
// 		Enabled:   true,
// 	}
// 	logEntry := log.WithField("test", "test")
// 	pipeline := New(config, logEntry)

// 	err := pipeline.encryptFiles(*config)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestOpenGPGEncryptionPX(t *testing.T) {

	providerConfig := &crypto.ProviderConfig{
		EncryptionKey: "/media/andmas/USB03/DDKeys/keys/PX/PX_UAT.asc",
	}
	providers := make(map[string]crypto.ProviderConfig, 1)
	providers["px"] = *providerConfig

	encryptConfig := &EncryptFilesConfig{
		SrcDir:    "/tmp/ddrun/Pickup/",
		OutputDir: "/tmp/ddrun/EncryptedPXTest/",
		Providers: providers,
		Enabled:   true,
	}
	logEntry := log.WithField("test", "test")

	ddConfig := &Config{}
	ddConfig.Tasks = &Tasks{
		EncryptFiles: encryptConfig,
	}

	pipeline := New(config, logEntry)

	err := pipeline.encryptFiles(*ddConfig)
	if err != nil {
		t.Error(err)
	}
}
