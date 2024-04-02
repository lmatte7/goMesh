package gomesh

import (
	"bytes"
	"flag"
	"testing"
	"time"

	pb "github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
)

var port = flag.String("port", "", "port the radio is connected to")

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

func TestGetChannelInfo(t *testing.T) {
	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	channel, err := radio.GetChannelInfo(0)
	if err != nil {
		t.Fatalf("Error when retrieving with channel info: %v", err)
	}

	if channel.Index != 0 {
		t.Fatalf("Error when retrieving channel: %v", err)
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

	err = radio.SetChannelURL("https://www.meshtastic.org/c/#ChASBGFzZGYaBGFzZGY6Aggg")
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

	chanSettings, err := radio.GetChannelInfo(0)
	if err != nil {
		t.Fatalf("Error retreiving channel settings: %v", err)
	}

	psk := []byte("asdf")
	if !bytes.Equal(chanSettings.Settings.Psk, psk) {
		t.Fatalf("Channel PSK not set correctly")
	}

}

func TestSetModem(t *testing.T) {

	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	err = radio.SetModemMode("vls")
	if err != nil {
		t.Fatalf("Error when communicating with radio: %v", err)
	}

}

func TestSetRadioConfig(t *testing.T) {
	radio, err := radioSetup()
	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}
	defer radio.Close()

	err = radio.SetRadioConfig("DebugLogEnabled", "True")
	if err != nil {
		t.Fatalf("Error setting config: %v", err)
	}

	time.Sleep(3 * time.Second)
	configPackets, _, err := radio.GetRadioConfig()

	if err != nil {
		t.Fatalf("Error when opening serial communications with radio: %v", err)
	}

	for _, config := range configPackets {

		if device := config.Config.GetDevice(); device != nil {
			if !device.DebugLogEnabled {
				t.Fatalf("Error setting config settings")
			}
		}
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

	time.Sleep(2 * time.Second)
	channel, err := radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.Settings.Name != chanName {
		t.Fatalf("Failed to add channel")
	}

	err = radio.DeleteChannel(1)
	if err != nil {
		t.Fatalf("Error Deleting channel: %v", err)
	}

	time.Sleep(2 * time.Second)
	channel, err = radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.Role != pb.Channel_DISABLED {
		t.Fatalf("Failed to delete channel")
	}

}

func TestSetChannelSettings(t *testing.T) {

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

	time.Sleep(2 * time.Second)
	channel, err := radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if channel.Settings.Name != chanName {
		t.Fatalf("Failed to add channel")
	}

	newPsk := "newPsk"
	err = radio.SetChannel(1, "Psk", newPsk)
	if err != nil {
		t.Fatalf("Error changing channel setting: %v", err)
	}

	time.Sleep(2 * time.Second)
	channel, err = radio.GetChannelInfo(1)
	if err != nil {
		t.Fatalf("Error retrieving channel: %v", err)
	}

	if string(channel.Settings.Psk) != newPsk {
		t.Fatalf("Failed to change channel")
	}

	time.Sleep(2 * time.Second)
	err = radio.DeleteChannel(1)
	if err != nil {
		t.Fatalf("Error Deleting channel: %v", err)
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
	err = radio.Init(*port)
	if err != nil {
		return Radio{}, err
	}

	return
}
