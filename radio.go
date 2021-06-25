package gomesh

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"
	pb "github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
	"google.golang.org/protobuf/proto"
)

const start1 = byte(0x94)
const start2 = byte(0xc3)
const headerLen = 4
const maxToFromRadioSzie = 512
const broadcastAddr = "^all"
const localAddr = "^local"
const defaultHopLimit = 3
const broadcastNum = 0xffffffff

// Radio holds the port and serial io.ReadWriteCloser struct to maintain one serial connection
type Radio struct {
	portNumber string
	serialPort io.ReadWriteCloser
}

// Init initializes the Serial connection for the radio
func (r *Radio) Init(serialPort string) error {
	r.portNumber = serialPort
	//Configure the serial port
	options := serial.OpenOptions{
		PortName:              r.portNumber,
		BaudRate:              921600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
		ParityMode:            serial.PARITY_NONE,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		return err
	}

	r.serialPort = port

	return nil
}

// sendPacket takes a protbuf packet, construct the appropriate header and sends it to the radio
func (r *Radio) sendPacket(protobufPacket []byte) (err error) {

	packageLength := len(string(protobufPacket))

	header := []byte{start1, start2, byte(packageLength>>8) & 0xff, byte(packageLength) & 0xff}

	radioPacket := append(header, protobufPacket...)
	_, err = r.serialPort.Write(radioPacket)
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	return

}

// ReadResponse reads any responses in the serial port, convert them to a FromRadio protobuf and return
func (r *Radio) ReadResponse() (FromRadioPackets []*pb.FromRadio, err error) {

	b := make([]byte, 1)

	emptyByte := make([]byte, 0)
	processedBytes := make([]byte, 0)
	repeatByteCounter := 0
	previousByte := make([]byte, 1)
	/************************************************************************************************
	* Process the returned data byte by byte until we have a valid command
	* Each command will come back with [START1, START2, PROTOBUF_PACKET]
	* where the protobuf packet is sent in binary. After reading START1 and START2
	* we use the next bytes to find the length of the packet.
	* After finding the length the looop continues to gather bytes until the length of the gathered
	* bytes is equal to the packet length plus the header
	 */
	for {
		_, err := r.serialPort.Read(b)
		// fmt.Printf("Byte: %q\n", b)
		if bytes.Equal(b, previousByte) {
			repeatByteCounter++
		} else {
			repeatByteCounter = 0
		}

		if err == io.EOF || repeatByteCounter > 20 {
			break
		} else if err != nil {
			return nil, err
		}
		copy(previousByte, b)

		if len(b) > 0 {

			pointer := len(processedBytes)

			processedBytes = append(processedBytes, b...)

			if pointer == 0 {
				if b[0] != start1 {
					processedBytes = emptyByte
				}
			} else if pointer == 1 {
				if b[0] != start2 {
					processedBytes = emptyByte
				}
			} else if pointer >= headerLen {
				packetLength := int((processedBytes[2] << 8) + processedBytes[3])

				if pointer == headerLen {
					if packetLength > maxToFromRadioSzie {
						processedBytes = emptyByte
					}
				}

				if len(processedBytes) != 0 && pointer+1 == packetLength+headerLen {
					fromRadio := pb.FromRadio{}
					if err := proto.Unmarshal(processedBytes[headerLen:], &fromRadio); err != nil {
						return nil, err
					}
					FromRadioPackets = append(FromRadioPackets, &fromRadio)
					processedBytes = emptyByte
				}
			}

		} else {
			break
		}

	}

	return FromRadioPackets, nil

}

// sendAdminPacket builds a admin message packet to send to the radio
func (r *Radio) createAdminPacket(nodeNum uint32, payload []byte) (packetOut []byte, err error) {

	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      nodeNum,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload:      payload,
						Portnum:      pb.PortNum_ADMIN_APP,
						WantResponse: true,
					},
				},
			},
		},
	}

	packetOut, err = proto.Marshal(&radioMessage)
	if err != nil {
		return nil, err
	}

	return

}

// getNodeNum returns the current NodeNumber after querying the radio
func (r *Radio) getNodeNum() (nodeNum uint32, err error) {
	// Send first request for Radio and Node information
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: 42}}

	out, err := proto.Marshal(&nodeInfo)
	if err != nil {
		return 0, err
	}

	r.sendPacket(out)

	radioResponses, err := r.ReadResponse()
	if err != nil {
		return 0, err
	}

	// Gather the Node number for channel settings requests
	nodeNum = 0
	for _, response := range radioResponses {
		if info, ok := response.GetPayloadVariant().(*pb.FromRadio_MyInfo); ok {
			nodeNum = info.MyInfo.MyNodeNum
		}
	}

	return
}

// GetRadioInfo retrieves information from the radio including config and adjacent Node information
func (r *Radio) GetRadioInfo() (radioResponses []*pb.FromRadio, err error) {
	// Send first request for Radio and Node information
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: 42}}

	out, err := proto.Marshal(&nodeInfo)
	if err != nil {
		return nil, err
	}

	r.sendPacket(out)

	radioResponses, err = r.ReadResponse()
	if err != nil {
		return nil, err
	}

	// Gather the Node number for channel settings requests
	var nodeNum uint32
	nodeNum = 0
	for _, response := range radioResponses {
		if info, ok := response.GetPayloadVariant().(*pb.FromRadio_MyInfo); ok {
			nodeNum = info.MyInfo.MyNodeNum
		}
	}

	// Send a second request to retrieve channel information
	channelInfo := pb.AdminMessage{
		Variant: &pb.AdminMessage_GetChannelRequest{
			GetChannelRequest: 1,
		},
	}

	out, err = proto.Marshal(&channelInfo)
	if err != nil {
		return nil, err
	}

	packetOut, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return nil, err
	}
	r.sendPacket(packetOut)

	newResponses, err := r.ReadResponse()
	if err != nil {
		return nil, err
	}

	radioResponses = append(radioResponses, newResponses...)

	return

}

// GetChannelInfo returns the current chanels settings for the radio
func (r *Radio) GetChannelInfo(channel int) (channelSettings pb.AdminMessage, err error) {

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return pb.AdminMessage{}, err
	}

	channel++
	channelInfo := pb.AdminMessage{
		Variant: &pb.AdminMessage_GetChannelRequest{
			GetChannelRequest: uint32(channel),
		},
	}

	out, err := proto.Marshal(&channelInfo)
	if err != nil {
		return pb.AdminMessage{}, err
	}

	if err != nil {
		return pb.AdminMessage{}, err
	}

	packetOut, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return pb.AdminMessage{}, err
	}
	r.sendPacket(packetOut)

	channelResponses, err := r.ReadResponse()
	if err != nil {
		return pb.AdminMessage{}, err
	}

	var channelPacket []byte
	for _, response := range channelResponses {
		if packet, ok := response.GetPayloadVariant().(*pb.FromRadio_Packet); ok {
			if packet.Packet.GetDecoded().GetPortnum() == pb.PortNum_ADMIN_APP {
				channelPacket = packet.Packet.GetDecoded().GetPayload()
			}
		}
	}

	if err := proto.Unmarshal(channelPacket, &channelSettings); err != nil {
		return pb.AdminMessage{}, err
	}

	return

}

func (r *Radio) GetRadioPreferences() (radioPreferences pb.AdminMessage, err error) {

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return pb.AdminMessage{}, err
	}

	radioPref := pb.AdminMessage{
		Variant: &pb.AdminMessage_GetRadioRequest{
			GetRadioRequest: true,
		},
	}

	out, err := proto.Marshal(&radioPref)
	if err != nil {
		return pb.AdminMessage{}, err
	}

	if err != nil {
		return pb.AdminMessage{}, err
	}

	packetOut, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return pb.AdminMessage{}, err
	}
	r.sendPacket(packetOut)

	channelResponses, err := r.ReadResponse()
	if err != nil {
		return pb.AdminMessage{}, err
	}

	var channelPacket []byte
	for _, response := range channelResponses {
		if packet, ok := response.GetPayloadVariant().(*pb.FromRadio_Packet); ok {
			if packet.Packet.GetDecoded().GetPortnum() == pb.PortNum_ADMIN_APP {
				channelPacket = packet.Packet.GetDecoded().GetPayload()
			}
		}
	}

	if err := proto.Unmarshal(channelPacket, &radioPreferences); err != nil {
		return pb.AdminMessage{}, err
	}

	return
}

// SendTextMessage sends a free form text message to other radios
func (r *Radio) SendTextMessage(message string, to int64) error {
	var address int64
	if to == 0 {
		address = broadcastNum
	} else {
		address = to
	}

	// This constant is defined in Constants_DATA_PAYLOAD_LEN, but not in a friendly way to use
	if len(message) > 240 {
		return errors.New("Message too large")
	}

	rand.Seed(time.Now().UnixNano())
	packetID := rand.Intn(2386828-1) + 1

	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      uint32(address),
				WantAck: true,
				Id:      uint32(packetID),
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload: []byte(message),
						Portnum: pb.PortNum_TEXT_MESSAGE_APP,
					},
				},
			},
		},
	}

	out, err := proto.Marshal(&radioMessage)
	if err != nil {
		return err
	}

	if err := r.sendPacket(out); err != nil {
		return err
	}

	return nil

}

// SetRadioOwner sets the owner of the radio visible on the public mesh
func (r *Radio) SetRadioOwner(name string) error {

	if len(name) <= 2 {
		return errors.New("Name too short")
	}

	adminPacket := pb.AdminMessage{
		Variant: &pb.AdminMessage_SetOwner{
			SetOwner: &pb.User{
				LongName:  name,
				ShortName: name[:3],
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return err
	}
	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil
}

// SetChannelURL sets the channel for the radio. The incoming channel should match the meshtastic URL format
// of a URL ending with /#{base_64_encoded_radio_params}
func (r *Radio) SetChannelURL(url string) error {

	// Split and unmarshel incoming base64 encoded protobuf packet
	split := strings.Split(url, "/#")
	channel := split[len(split)-1]

	cDec, err := base64.RawURLEncoding.DecodeString(channel)
	if err != nil {
		return err
	}
	encChannels := pb.ChannelSet{}

	if err := proto.Unmarshal(cDec, &encChannels); err != nil {
		return err
	}

	// The default pre-shared key
	// PskByte := []byte{0xd4, 0xf1, 0xbb, 0x3a, 0x20, 0x29, 0x07, 0x59, 0xf0, 0xbc, 0xff, 0xab, 0xcf, 0x4e, 0x69, 0xbf}

	var protoChannel *pb.ChannelSettings
	for i, cSet := range encChannels.Settings {
		protoChannel = cSet

		var role pb.Channel_Role
		if i == 0 {
			role = pb.Channel_PRIMARY
		} else {
			role = pb.Channel_SECONDARY
		}

		// Send settings to Radio
		adminPacket := pb.AdminMessage{
			Variant: &pb.AdminMessage_SetChannel{
				SetChannel: &pb.Channel{
					Index: int32(i),
					Role:  role,
					Settings: &pb.ChannelSettings{
						Psk:         protoChannel.Psk,
						ModemConfig: protoChannel.ModemConfig,
					},
				},
			},
		}

		out, err := proto.Marshal(&adminPacket)
		if err != nil {
			return err
		}

		nodeNum, err := r.getNodeNum()
		if err != nil {
			return err
		}
		packet, err := r.createAdminPacket(nodeNum, out)
		if err != nil {
			return err
		}

		if err := r.sendPacket(packet); err != nil {
			return err
		}
	}

	return nil
}

// SetModemMode sets the channel modem setting to be fast or slow
func (r *Radio) SetModemMode(channel int) error {

	var modemSetting int

	if channel == 0 {
		modemSetting = int(pb.ChannelSettings_Bw125Cr48Sf4096)
	} else {
		modemSetting = int(pb.ChannelSettings_Bw500Cr45Sf128)
	}

	chanSettings, err := r.GetChannelInfo(1)
	if err != nil {
		return err
	}

	// Send settings to Radio
	adminPacket := pb.AdminMessage{
		Variant: &pb.AdminMessage_SetChannel{
			SetChannel: &pb.Channel{
				Index: 0,
				Role:  chanSettings.GetGetChannelResponse().GetRole(),
				Settings: &pb.ChannelSettings{
					Psk:         chanSettings.GetGetChannelResponse().GetSettings().GetPsk(),
					ModemConfig: pb.ChannelSettings_ModemConfig(modemSetting),
				},
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return err
	}
	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil

}

// AddChannel adds a new channel to the radio
func (r *Radio) AddChannel(name string, cIndex int) error {

	var role pb.Channel_Role
	if cIndex == 0 {
		role = pb.Channel_PRIMARY
	} else {
		role = pb.Channel_SECONDARY
	}

	// Send settings to Radio
	adminPacket := pb.AdminMessage{
		Variant: &pb.AdminMessage_SetChannel{
			SetChannel: &pb.Channel{
				Index: int32(cIndex),
				Role:  role,
				Settings: &pb.ChannelSettings{
					Psk:  genPSK256(),
					Name: name,
				},
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return err
	}
	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil

}

func (r *Radio) DeleteChannel(cIndex int) error {

	channelInfo, err := r.GetChannelInfo(cIndex)
	if err != nil {
		return err
	}

	if channelInfo.GetGetChannelResponse().GetRole() == pb.Channel_PRIMARY {
		return errors.New("cannot delete PRIMARY channel")
	}

	// Send settings to Radio
	adminPacket := pb.AdminMessage{
		Variant: &pb.AdminMessage_SetChannel{
			SetChannel: &pb.Channel{
				Index:    int32(cIndex),
				Role:     pb.Channel_DISABLED,
				Settings: nil,
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return err
	}
	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil

}

// SetChannel sets one of two channels for the radio
func (r *Radio) SetChannel(chIndex int, key string, value string) error {

	channel, err := r.GetChannelInfo(chIndex)
	if err != nil {
		return err
	}

	if channel.GetGetChannelResponse().GetRole() == pb.Channel_DISABLED {
		return errors.New("no channel for provided index")
	}

	channelSettings := channel.GetGetChannelResponse().GetSettings()
	rPref := reflect.ValueOf(channelSettings)

	rPref = rPref.Elem()

	fv := rPref.FieldByName(key)
	if !fv.IsValid() {
		return errors.New("unknown Field")
	}

	if !fv.CanSet() {
		return errors.New("unknown Field")
	}

	vType := fv.Type()

	// The acceptable values that can be set from the command line are uint32 and bool, so only check for those
	switch vType.Kind() {
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		fv.SetBool(boolValue)
	case reflect.Uint32:
		uintValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return err
		}
		fv.SetUint(uintValue)
	case reflect.Array:
		arrayValue := []byte(value)
		fv.SetBytes(arrayValue)
	case reflect.String:
		fv.SetString(value)
	}

	// Send settings to Radio
	adminPacket := pb.AdminMessage{
		Variant: &pb.AdminMessage_SetChannel{
			SetChannel: &pb.Channel{
				Index:    int32(chIndex),
				Role:     channel.GetGetChannelResponse().GetRole(),
				Settings: channelSettings,
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return err
	}
	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil

}

// SetUserPreferences allows an freeform setting of values in the RadioConfig_UserPreferences struct
func (r *Radio) SetUserPreferences(key string, value string) error {

	preferences, err := r.GetRadioPreferences()
	if err != nil {
		return err
	}

	rPref := reflect.ValueOf(preferences.GetGetRadioResponse().GetPreferences())

	rPref = rPref.Elem()

	fv := rPref.FieldByName(key)
	if !fv.IsValid() {
		return errors.New("unknown Field")
	}

	if !fv.CanSet() {
		return errors.New("unknown Field")
	}

	vType := fv.Type()

	// The acceptable values that can be set from the command line are uint32 and bool, so only check for those
	switch vType.Kind() {
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		fv.SetBool(boolValue)
	case reflect.Uint32:
		uintValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return err
		}
		fv.SetUint(uintValue)
	case reflect.Int32:
		intValue, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return err
		}
		fv.SetInt(intValue)
	case reflect.String:
		fv.SetString(value)

	}

	// Send settings to Radio
	adminPacket := pb.AdminMessage{
		Variant: &pb.AdminMessage_SetRadio{
			SetRadio: &pb.RadioConfig{
				Preferences: preferences.GetGetRadioResponse().GetPreferences(),
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return err
	}
	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil
}

// SetLocation sets a fixed location for the radio
func (r *Radio) SetLocation(lat float64, long float64, alt int32) error {

	positionPacket := pb.Position{
		LatitudeI:  int32(lat),
		LongitudeI: int32(long),
		Altitude:   int32(alt),
		Time:       0,
	}

	out, err := proto.Marshal(&positionPacket)
	if err != nil {
		return err
	}

	nodeNum, err := r.getNodeNum()
	if err != nil {
		return err
	}

	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      nodeNum,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload:      out,
						Portnum:      pb.PortNum_POSITION_APP,
						WantResponse: true,
					},
				},
			},
		},
	}

	packet, err := proto.Marshal(&radioMessage)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil
}

// Close closes the serial port. Added so users can defer the close after opening
func (r *Radio) Close() {
	if r.serialPort != nil {
		r.serialPort.Close()
	}
}
