package gomesh

import (
	"encoding/base64"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
	pb "github.com/lmatte7/gomesh/github.com/meshtastic/gomeshproto"
)

// GetChannelInfo returns the current chanels settings for the radio
func (r *Radio) GetChannels() (channels []*pb.Channel, err error) {

	checks := 0

	for checks < 5 && len(channels) == 0 {
		info, err := r.GetRadioInfo()
		if err != nil {
			return nil, err
		}

		for _, packet := range info {
			if channelInfo, ok := packet.GetPayloadVariant().(*gomeshproto.FromRadio_Channel); ok {
				channels = append(channels, channelInfo.Channel)
			}
		}

		// If we didn't get any channels wait and try again
		time.Sleep(50 * time.Millisecond)
		checks++
	}

	if len(channels) == 0 {
		return nil, errors.New("no channels found")
	}

	return channels, nil
}

// GetChannelInfo returns the current chanels settings for the radio
func (r *Radio) GetChannelInfo(index int) (channelSettings *pb.Channel, err error) {

	info, err := r.GetChannels()
	if err != nil {
		return &pb.Channel{}, err
	}

	for _, channel := range info {
		if channel.Index == int32(index) {
			return channel, nil
		}
	}

	return &pb.Channel{}, errors.New("channel not found")
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
			PayloadVariant: &pb.AdminMessage_SetChannel{
				SetChannel: &pb.Channel{
					Index: int32(i),
					Role:  role,
					Settings: &pb.ChannelSettings{
						Psk:            protoChannel.Psk,
						ModuleSettings: protoChannel.ModuleSettings,
					},
				},
			},
		}

		out, err := proto.Marshal(&adminPacket)
		if err != nil {
			return err
		}

		nodeNum := r.nodeNum

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

// AddChannel adds a new channel to the radio
func (r *Radio) AddChannel(name string, cIndex int) error {

	var role pb.Channel_Role
	if cIndex == 0 {
		role = pb.Channel_PRIMARY
	} else {
		role = pb.Channel_SECONDARY
	}

	// Grab the channel and check if it's disabled, if not return an error
	curChannel, err := r.GetChannelInfo(cIndex)
	if err != nil {
		return errors.New("error getting channel info")
	}

	if curChannel.Role != pb.Channel_DISABLED {
		return errors.New("channel already exists")
	}

	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetChannel{
			SetChannel: &pb.Channel{
				Index: int32(cIndex),
				Role:  role,
				Settings: &pb.ChannelSettings{
					Psk:            genPSK256(),
					Name:           name,
					ModuleSettings: curChannel.Settings.ModuleSettings,
				},
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum := r.nodeNum

	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil

}

// SetChannel sets a channel value
func (r *Radio) SetChannel(chIndex int, key string, value string) error {

	channel, err := r.GetChannelInfo(chIndex)
	if err != nil {
		return err
	}

	if channel.Role == pb.Channel_DISABLED {
		return errors.New("no channel for provided index")
	}

	channelSettings := channel.GetSettings()

	rPref := reflect.ValueOf(channelSettings)

	rPref = rPref.Elem()

	fv := rPref.FieldByName(key)

	if fv.IsValid() && fv.CanSet() {
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
			uintValue, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return err
			}
			fv.SetInt(uintValue)
		case reflect.Array:
			arrayValue := []byte(value)
			fv.SetBytes(arrayValue)
		case reflect.String:
			fv.SetString(value)
		case reflect.Slice:
			fv.SetBytes([]byte(value))
		}

	} else if key == "PositionPrecision" {
		uintValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return err
		}
		channelSettings.ModuleSettings = &pb.ModuleSettings{
			PositionPrecision: uint32(uintValue),
		}
	} else {
		return errors.New("unknown Field")
	}

	// Send settings to Radio
	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetChannel{
			SetChannel: &pb.Channel{
				Index:    int32(chIndex),
				Role:     channel.Role,
				Settings: channelSettings,
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	nodeNum := r.nodeNum

	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil

}

// Delete a channel from the radio
func (r *Radio) DeleteChannel(cIndex int) error {

	channelInfo, err := r.GetChannelInfo(cIndex)
	if err != nil {
		return err
	}

	if channelInfo.Role == pb.Channel_PRIMARY {
		return errors.New("cannot delete PRIMARY channel")
	}

	// Send settings to Radio
	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetChannel{
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

	nodeNum := r.nodeNum

	packet, err := r.createAdminPacket(nodeNum, out)
	if err != nil {
		return err
	}

	if err := r.sendPacket(packet); err != nil {
		return err
	}

	return nil

}
