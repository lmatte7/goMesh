package gomesh

import (
	"bytes"
	"testing"

	pb "github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
)

func TestRadioInfo(t *testing.T) {

	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

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
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	err = radio.SendTextMessage("Test", 0)

	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

}

func TestSetOwner(t *testing.T) {

	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	err = radio.SetRadioOwner("Test Owner")
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	radioResponses, err := radio.GetRadioInfo()
	if err != nil {
		t.Fatalf("Error when retrieving radio information: %v", err)
	}
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
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	err = radio.SetChannelURL("https://www.meshtastic.org/d/#CgUYAyIBAQ")
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	chanSettings, err := radio.GetChannelInfo(0)
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
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	err = radio.SetModemMode(1)
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	chanSettings, err := radio.GetChannelInfo(0)
	if err != nil {
		t.Fatalf("Error retreiving channel settings: %v", err)
	}

	if chanSettings.GetGetChannelResponse().GetSettings().GetModemConfig() != pb.ChannelSettings_Bw500Cr45Sf128 {
		t.Fatalf("Channel modem not set correctly")
	}
	err = radio.SetModemMode(0)
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

}

func TestSetRadioPref(t *testing.T) {
	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	err = radio.SetUserPreferences("SendOwnerInterval", "20")
	if err != nil {
		t.Fatalf("Error setting preference: %v", err)
	}

	// time.Sleep(3 * time.Second)
	radioPrefs, err := radio.GetRadioPreferences()

	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}

	if radioPrefs.GetGetRadioResponse().Preferences.GetSendOwnerInterval() != 20 {
		t.Fatalf("Radio Preference Not Set Correctly")
	}
	err = radio.SetUserPreferences("SendOwnerInterval", "0")
	if err != nil {
		t.Fatalf("Error setting preference: %v", err)
	}

}

func TestAddDeleteChannel(t *testing.T) {
	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	chanName := "test"

	err = radio.AddChannel(chanName, 1)
	if err != nil {
		t.Fatalf("Error adding channel: %v", err)
	}

	channel, err := radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.GetGetChannelResponse().GetSettings().GetName() != chanName {
		t.Fatalf("Failed to add channel")
	}

	err = radio.DeleteChannel(1)
	if err != nil {
		t.Fatalf("Error Deleting channel: %v", err)
	}

	channel, err = radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.GetGetChannelResponse().GetRole() != pb.Channel_DISABLED {
		t.Fatalf("Failed to delete channel")
	}

}

func TestSetChannel(t *testing.T) {

	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	chanName := "test"

	err = radio.AddChannel(chanName, 1)
	if err != nil {
		t.Fatalf("Error adding channel: %v", err)
	}

	channel, err := radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.GetGetChannelResponse().GetSettings().GetName() != chanName {
		t.Fatalf("Failed to add channel")
	}

	newName := "new"
	err = radio.SetChannel(1, "Name", newName)
	if err != nil {
		t.Fatalf("Error changing channel setting: %v", err)
	}

	channel, err = radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.GetGetChannelResponse().GetSettings().GetName() != newName {
		t.Fatalf("Failed to change channel")
	}

	err = radio.DeleteChannel(1)
	if err != nil {
		t.Fatalf("Error Deleting channel: %v", err)
	}

	channel, err = radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.GetGetChannelResponse().GetRole() != pb.Channel_DISABLED {
		t.Fatalf("Failed to delete channel")
	}

}

func TestSetLocation(t *testing.T) {

	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	latitude := 32.048164
	err = radio.SetLocation(latitude, -87.581624, 127)
	if err != nil {
		t.Fatalf("Error setting location: %v", err)
	}

	radioResponses, err := radio.GetRadioInfo()
	if err != nil {
		t.Fatalf("Error retreiving radio information: %v", err)
	}

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
			if node.NodeInfo.GetPosition().GetLatitudeI() != int32(latitude) {
				t.Fatalf("Location not set correctly")
			}
		}
	}

}

func radioSetup() (radio Radio, err error) {
	radio = Radio{}
	// err = radio.Init("192.168.86.40")
	err = radio.Init("/dev/cu.SLAB_USBtoUART")
	if err != nil {
		return Radio{}, err
	}
	return
}
