package gomesh

import (
	"bytes"
	"testing"

	pb "github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
)

func TestRadioInfo(t *testing.T) {

	radio, err := radioSetup()
	defer radio.Close()

	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	radioResponses, err := radio.GetRadioInfo()
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	if len(radioResponses) < 4 {
		t.Fatalf("Missing Results")
	}

}

func TestSendText(t *testing.T) {

	radio, err := radioSetup()
	defer radio.Close()

	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	err = radio.SendTextMessage("Test", 0)

	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

}

func TestSetOwner(t *testing.T) {

	radio, err := radioSetup()
	defer radio.Close()

	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	err = radio.SetRadioOwner("Test Owner")
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	radioResponses, err := radio.GetRadioInfo()
	nodes := make([]*pb.FromRadio_NodeInfo, 0)
	var nodeNum uint32
	nodeNum = 0

	for _, response := range radioResponses {

		if nodeInfo, ok := response.GetPayloadVariant().(*pb.FromRadio_NodeInfo); ok {
			nodes = append(nodes, nodeInfo)
		}

		if info, ok := response.GetPayloadVariant().(*pb.FromRadio_MyInfo); ok {
			nodeNum = info.MyInfo.MyNodeNum
		}

	}

	for _, node := range nodes {
		if node.NodeInfo.Num == nodeNum {
			if node.NodeInfo.User.LongName != "Test Owner" {
				t.Fatalf("Owner name not correctly set")
			}
		}
	}
	// If test succeeds change name for future tests
	err = radio.SetRadioOwner("Owner")
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

}

func TestSetChannelURL(t *testing.T) {

	radio, err := radioSetup()
	defer radio.Close()

	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}

	err = radio.SetChannelURL("https://www.meshtastic.org/d/#CgUYAyIBAQ")
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	chanSettings, err := radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retreiving channel settings: %v", err)
	}

	psk := []byte("\001")
	if bytes.Compare(chanSettings.GetGetChannelResponse().GetSettings().GetPsk(), psk) != 0 {
		t.Fatalf("Channel PSK not set correctly")
	}

	if chanSettings.GetGetChannelResponse().GetSettings().GetModemConfig() != pb.ChannelSettings_Bw125Cr48Sf4096 {
		t.Fatalf("Channel modem not set correctly")
	}

}

func TestSetModem(t *testing.T) {

	radio, err := radioSetup()
	defer radio.Close()

	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}

	err = radio.SetModemMode(1)
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	// TODO: Finish verification for this test
	chanSettings, err := radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retreiving channel settings: %v", err)
	}

	psk := []byte("\001")
	if bytes.Compare(chanSettings.GetGetChannelResponse().GetSettings().GetPsk(), psk) != 0 {
		t.Fatalf("Channel PSK not set correctly")
	}

	if chanSettings.GetGetChannelResponse().GetSettings().GetModemConfig() != pb.ChannelSettings_Bw125Cr48Sf4096 {
		t.Fatalf("Channel modem not set correctly")
	}

}

func radioSetup() (radio Radio, err error) {
	radio = Radio{}
	err = radio.Init("/dev/cu.SLAB_USBtoUART")
	if err != nil {
		return Radio{}, err
	}
	return
}
