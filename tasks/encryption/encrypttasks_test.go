package encryption

import (
	"testing"

	"github.com/masenocturnal/pipefire/internal/crypto"
)

func TestOpenGPGEncryptionPX(t *testing.T) {

	providerConfig := &crypto.ProviderConfig{
		EncryptionKey: "/media/andmas/USB03/DDKeys/keys/PX/PX_UAT.asc",
	}
	providers := make(map[string]crypto.ProviderConfig, 1)
	providers["px"] = *providerConfig

	encryptConfig := &crypto.EncryptFilesConfig{
		SrcDir:    "/tmp/ddrun/Pickup/",
		OutputDir: "/tmp/ddrun/EncryptedPXTest/",
		Providers: providers,
		Enabled:   true,
	}
	_ = encryptConfig

	// tasksConfig := &TasksConfig{
	// 	EncryptFiles: encryptConfig,
	// }

	// ddConfig := &PipelineConfig{}
	// ddConfig.Tasks = tasksConfig

	// pipeline, err := New(ddConfig)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// errs := encryption.PGPEncryptFilesForTransfer(encryptConfig, log.NewEntry(log.StandardLogger()))
	// if len(errs) > 0 {
	// 	for err := range errs {
	// 		t.Error(err)
	// 	}

	// }
}
