# Go Mesh

Go Mesh (goMesh) is a package that provides prebuilt methods and tools for interacting with [meshtastic](https://meshtastic.org/) radios and is compatible with Windows, Linux and Mac. The only requirements for this tool are the ESP32 [drivers](https://www.silabs.com/developers/usb-to-uart-bridge-vcp-drivers) if not already installed.

## Initialization

All communications with meshtastic radios can be done through the `Radio` struct. Below is an example that shows the setup:

```
radio := Radio{}
radio.Init("/dev/cu.SLAB_USBtoUART")
defer radio.Close()
```

This example uses the Mac port name from the ESP32 drivers
(`cu.SLAB_USBtoUART`) but this will change depending on the OS and the drivers used. It is possible to communicate with meshtastic radios over TCP as well. Passing in an IP address will automatically have the Radio client choose TCP communications. 

Remember to `defer` radio.Close() to close the port that's being used to communicate with the device.

## Usage

There are multiple available functions to interact with the radios and perform different functions.

```
r := gomesh.Radio{}
responses, err := r.GetRadioInfo()
if err != nil {
  return err
})
```

In this example the responses could then be read and processed by type to process the data.

```
for _, response := range responses {
  if info, ok := response.GetPayloadVariant().(*pb.FromRadio_MyInfo); ok {
    // Do something with the response
  }
}
```

## Tests

Tests for each major radio function are provided in `radio_test.go`. The full test suite should be run while the machine running the tests is plugged into a meshtastic radio. The test require a command line argument that specifies the port a Meshtastic radio is connected to. 
To run all test use:  
```
go test -args -port=/dev/cu.usbserial-0200674E
```

To run a single test specify the test to run and use the command: 
```
go test -run TestSetChannelURL -args -port=/dev/cu.usbserial-0200674E
```

## Feedback

This package is still under development. Any issues or feedback can be submitted to the [issues](https://github.com/lmatte7/goMesh/issues) page.
