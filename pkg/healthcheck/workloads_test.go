package healthcheck

import (
	"testing"

	"github.com/openshift/microshift/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_csiComponentsAreExpected(t *testing.T) {
	testData := []struct {
		name           string
		cfg            config.Config
		expectedResult []string
	}{
		{
			name: "empty",
			cfg: config.Config{Storage: config.Storage{
				OptionalCSIComponents: []config.OptionalCsiComponent{},
			}},
			expectedResult: []string{"csi-snapshot-controller"},
		},
		{
			name: "none",
			cfg: config.Config{Storage: config.Storage{
				OptionalCSIComponents: []config.OptionalCsiComponent{config.CsiComponentNone},
			}},
			expectedResult: nil,
		},
		{
			name: "only controller",
			cfg: config.Config{Storage: config.Storage{
				OptionalCSIComponents: []config.OptionalCsiComponent{config.CsiComponentSnapshot},
			}},
			expectedResult: []string{"csi-snapshot-controller"},
		},
	}
	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			cfg := td.cfg
			result := getExpectedCSIComponents(&cfg)
			assert.Equal(t, td.expectedResult, result)
		})
	}
}
