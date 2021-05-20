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
(`cu.SLAB_USBtoUART`) but this will change depending on the OS and the drivers used. Remember to `defer` radio.Close() to close the port that's being used to communicate with the device.

## Usage
There are multiple available functions to interact with the radios and perform different functions.

```
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

## Feedback
This package is still under development. Any issues or feedback can be submitted to the [issues](https://github.com/lmatte7/meshGo/issues) page.