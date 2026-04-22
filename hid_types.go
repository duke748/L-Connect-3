package main

// hidDeviceEntry holds metadata for one HID device returned by hid_enumerate.
// Defined in a shared file so both windows and non-windows builds compile.
type hidDeviceEntry struct {
	VendorID           uint16
	ProductID          uint16
	UsagePage          uint16
	Usage              uint16
	InterfaceNumber    int
	ManufacturerString string
	ProductString      string
	SerialNumber       string
}
