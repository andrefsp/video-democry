package chap7

/*
import (
	"errors"
	"fmt"

	"github.com/pion/webrtc/v3"
)

func getPayloadType(m webrtc.MediaEngine, codecType webrtc.RTPCodecType, codecName string) (uint8, error) {
	for _, codec := range m.GetCodecsByKind(codecType) {
		if codec.Name == codecName {
			return codec.PayloadType, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("Remote peer does not support %s", codecName))
}

*/
