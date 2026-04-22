//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

// hidBridge wraps the Windows hidapi.dll library, providing access to HID device functions
// for communicating with Lian-Li L-Connect 3 devices. It lazily loads the DLL and caches
// procedure pointers for each supported hidapi function (init, open, write, read, close, exit).
// This allows communication with connected fans and RGB lighting controllers via HID reports.
type hidBridge struct {
	dll           *syscall.LazyDLL  // Lazy-loaded hidapi.dll
	initProc      *syscall.LazyProc // hid_init() - Initialize hidapi library
	openProc      *syscall.LazyProc // hid_open(vendorID, productID, ...) - Open device by VID/PID
	writeProc     *syscall.LazyProc // hid_write(handle, buf, len) - Send HID report to device
	getInputProc  *syscall.LazyProc // hid_get_input_report(handle, buf, len) - Receive input report (optional)
	closeProc     *syscall.LazyProc // hid_close(handle) - Close device handle
	exitProc      *syscall.LazyProc // hid_exit() - Cleanup hidapi library
	enumerateProc *syscall.LazyProc // hid_enumerate(vid, pid) - List connected HID devices (optional)
	freeEnumProc  *syscall.LazyProc // hid_free_enumeration(devs) - Free device info list (optional)
}

// hidProbe verifies that hidapi can open the target VID/PID.
func hidProbe(vendorID uint16, productID uint16) (string, error) {
	bridge, err := newHIDBridge()
	if err != nil {
		return "", err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return "", err
	}
	defer bridge.close(handle)

	return fmt.Sprintf("hid probe ok: opened device VID=0x%04X PID=0x%04X", vendorID, productID), nil
}

// hidFanSet sends the observed fan-speed write frame and commit frame.
func hidFanSet(vendorID uint16, productID uint16, port int, speed int) error {
	bridge, err := newHIDBridge()
	if err != nil {
		return err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return err
	}
	defer bridge.close(handle)

	portCmd := byte(0x1F + port)
	if err := bridge.write(handle, []byte{0xE0, portCmd, 0x00, byte(speed), 0x00, 0x00, 0x00}); err != nil {
		return fmt.Errorf("send fan speed command: %w", err)
	}
	if err := bridge.write(handle, []byte{0xE0, 0x50, 0x00, 0x00, 0x00, 0x00, 0x00}); err != nil {
		return fmt.Errorf("send commit command: %w", err)
	}

	return nil
}

// hidSetStaticColorAll applies static color to primary channels 0,2,4,6.
func hidSetStaticColorAll(vendorID uint16, productID uint16, red uint8, green uint8, blue uint8, brightnessPct int) error {
	bridge, err := newHIDBridge()
	if err != nil {
		return err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return err
	}
	defer bridge.close(handle)

	for channel := 0; channel < 8; channel += 2 {
		if err := hidSetStaticColorChannel(bridge, handle, channel, red, green, blue, brightnessPct); err != nil {
			return fmt.Errorf("set color failed for channel %d: %w", channel, err)
		}
	}

	return nil
}

// hidSetStaticColorPort applies static color to a single visible port using one channel.
func hidSetStaticColorPort(vendorID uint16, productID uint16, port int, red uint8, green uint8, blue uint8, brightnessPct int) error {
	if port < 1 || port > 4 {
		return fmt.Errorf("port must be 1..4")
	}

	bridge, err := newHIDBridge()
	if err != nil {
		return err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return err
	}
	defer bridge.close(handle)

	channel := (port - 1) * 2
	if err := hidSetStaticColorChannel(bridge, handle, channel, red, green, blue, brightnessPct); err != nil {
		return fmt.Errorf("set color failed for port %d (channel %d): %w", port, channel, err)
	}

	return nil
}

// hidSetStaticColorChannelByID applies static color to one raw SL Infinity channel.
func hidSetStaticColorChannelByID(vendorID uint16, productID uint16, channel int, red uint8, green uint8, blue uint8, brightnessPct int) error {
	if channel < 0 || channel > 7 {
		return fmt.Errorf("channel must be 0..7")
	}

	bridge, err := newHIDBridge()
	if err != nil {
		return err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return err
	}
	defer bridge.close(handle)

	if err := hidSetStaticColorChannel(bridge, handle, channel, red, green, blue, brightnessPct); err != nil {
		return fmt.Errorf("set color failed for channel %d: %w", channel, err)
	}

	return nil
}

func hidApplyEffectChannelByID(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, hasColor bool, red uint8, green uint8, blue uint8) error {
	if channel < 0 || channel > 7 {
		return fmt.Errorf("channel must be 0..7")
	}

	bridge, err := newHIDBridge()
	if err != nil {
		return err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return err
	}
	defer bridge.close(handle)

	if hasColor {
		if err := bridge.sendStartAction(handle, channel, 4); err != nil {
			return fmt.Errorf("start action failed: %w", err)
		}
		ledData := buildStaticColorLEDData(red, green, blue, brightnessPct)
		if err := bridge.sendColorData(handle, channel, 80, ledData); err != nil {
			return fmt.Errorf("color data failed: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	brightness := hidBrightnessCode(brightnessPct)
	if err := bridge.sendCommitAction(handle, channel, effect, byte(speed), byte(direction), brightness); err != nil {
		return fmt.Errorf("commit action failed: %w", err)
	}

	time.Sleep(10 * time.Millisecond)
	return nil
}

func hidApplyEffectPaletteChannelByID(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, colors []effectColor) error {
	if channel < 0 || channel > 7 {
		return fmt.Errorf("channel must be 0..7")
	}

	bridge, err := newHIDBridge()
	if err != nil {
		return err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return err
	}
	defer bridge.close(handle)

	if len(colors) > 0 {
		if err := bridge.sendStartAction(handle, channel, 4); err != nil {
			return fmt.Errorf("start action failed: %w", err)
		}
		ledData := buildPaletteLEDData(colors, brightnessPct)
		if err := bridge.sendColorData(handle, channel, 80, ledData); err != nil {
			return fmt.Errorf("color data failed: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	brightness := hidBrightnessCode(brightnessPct)
	if err := bridge.sendCommitAction(handle, channel, effect, byte(speed), byte(direction), brightness); err != nil {
		return fmt.Errorf("commit action failed: %w", err)
	}

	time.Sleep(10 * time.Millisecond)
	return nil
}

func hidSetStaticColorChannel(bridge *hidBridge, handle uintptr, channel int, red uint8, green uint8, blue uint8, brightnessPct int) error {
	brightness := hidBrightnessCode(brightnessPct)
	ledData := buildStaticColorLEDData(red, green, blue, brightnessPct)

	if err := bridge.sendStartAction(handle, channel, 4); err != nil {
		return fmt.Errorf("start action failed: %w", err)
	}

	if err := bridge.sendColorData(handle, channel, 80, ledData); err != nil {
		return fmt.Errorf("color data failed: %w", err)
	}

	time.Sleep(10 * time.Millisecond)

	if err := bridge.sendCommitAction(handle, channel, 0x01, 0x00, 0x00, brightness); err != nil {
		return fmt.Errorf("commit action failed: %w", err)
	}

	time.Sleep(10 * time.Millisecond)
	return nil
}

func hidReadRPM(vendorID uint16, productID uint16) ([4]uint16, error) {
	var rpmByPort [4]uint16

	bridge, err := newHIDBridge()
	if err != nil {
		return rpmByPort, err
	}

	handle, err := bridge.open(vendorID, productID)
	if err != nil {
		return rpmByPort, err
	}
	defer bridge.close(handle)

	report, err := bridge.getInputReport(handle, 0xE0)
	if err != nil {
		return rpmByPort, fmt.Errorf("read input report failed: %w", err)
	}

	rpmByPort[0] = (uint16(report[1]) << 8) | uint16(report[2])
	rpmByPort[1] = (uint16(report[3]) << 8) | uint16(report[4])
	rpmByPort[2] = (uint16(report[5]) << 8) | uint16(report[6])
	rpmByPort[3] = (uint16(report[7]) << 8) | uint16(report[8])

	return rpmByPort, nil
}

// hidDeviceEntry holds the parsed metadata for one HID device returned by hid_enumerate.
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

// hid_device_info struct field offsets for Windows x64 (MSVC ABI).
//
// C layout:
//
//	offset  0: char*          path
//	offset  8: uint16         vendor_id
//	offset 10: uint16         product_id
//	offset 12: [4 pad]
//	offset 16: wchar_t*       serial_number
//	offset 24: uint16         release_number
//	offset 26: [6 pad]
//	offset 32: wchar_t*       manufacturer_string
//	offset 40: wchar_t*       product_string
//	offset 48: uint16         usage_page
//	offset 50: uint16         usage
//	offset 52: int32          interface_number
//	offset 56: hid_device_info* next
const (
	hidInfoOffVendorID           uintptr = 8
	hidInfoOffProductID          uintptr = 10
	hidInfoOffSerialNumber       uintptr = 16
	hidInfoOffManufacturerString uintptr = 32
	hidInfoOffProductString      uintptr = 40
	hidInfoOffUsagePage          uintptr = 48
	hidInfoOffUsage              uintptr = 50
	hidInfoOffInterfaceNumber    uintptr = 52
	hidInfoOffNext               uintptr = 56
)

func hidReadUint16At(base uintptr, offset uintptr) uint16 {
	return *(*uint16)(unsafe.Pointer(base + offset))
}

func hidReadUintptrAt(base uintptr, offset uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(base + offset))
}

func hidReadInt32At(base uintptr, offset uintptr) int32 {
	return *(*int32)(unsafe.Pointer(base + offset))
}

// hidReadWcharString reads a null-terminated UTF-16 string from a wchar_t pointer.
func hidReadWcharString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	var buf []uint16
	for i := uintptr(0); ; i += 2 {
		c := *(*uint16)(unsafe.Pointer(ptr + i))
		if c == 0 {
			break
		}
		buf = append(buf, c)
	}
	return syscall.UTF16ToString(buf)
}

// hidEnumerateDevices calls hid_enumerate(0, 0) to list every connected HID device
// and returns their metadata. Useful for discovering VID/PID values for unknown devices.
func hidEnumerateDevices() ([]hidDeviceEntry, error) {
	bridge, err := newHIDBridge()
	if err != nil {
		return nil, err
	}
	if bridge.enumerateProc == nil {
		return nil, fmt.Errorf("hid_enumerate is not available in the current hidapi.dll")
	}

	// Passing 0,0 enumerates all HID devices regardless of VID/PID.
	head, _, callErr := bridge.enumerateProc.Call(0, 0)
	if head == 0 {
		if callErr == syscall.Errno(0) {
			return nil, nil // no devices found
		}
		return nil, fmt.Errorf("hid_enumerate failed: %w", callErr)
	}
	if bridge.freeEnumProc != nil {
		defer bridge.freeEnumProc.Call(head)
	}

	var entries []hidDeviceEntry
	for node := head; node != 0; node = hidReadUintptrAt(node, hidInfoOffNext) {
		entries = append(entries, hidDeviceEntry{
			VendorID:           hidReadUint16At(node, hidInfoOffVendorID),
			ProductID:          hidReadUint16At(node, hidInfoOffProductID),
			UsagePage:          hidReadUint16At(node, hidInfoOffUsagePage),
			Usage:              hidReadUint16At(node, hidInfoOffUsage),
			InterfaceNumber:    int(hidReadInt32At(node, hidInfoOffInterfaceNumber)),
			ManufacturerString: hidReadWcharString(hidReadUintptrAt(node, hidInfoOffManufacturerString)),
			ProductString:      hidReadWcharString(hidReadUintptrAt(node, hidInfoOffProductString)),
			SerialNumber:       hidReadWcharString(hidReadUintptrAt(node, hidInfoOffSerialNumber)),
		})
	}
	return entries, nil
}

func newHIDBridge() (*hidBridge, error) {
	candidates := []string{
		"hidapi.dll",
		filepath.Join("C:/Program Files/Lian-Li/L-Connect 3", "hidapi.dll"),
	}

	var lastErr error
	for _, dllPath := range candidates {
		dll := syscall.NewLazyDLL(dllPath)
		initProc := dll.NewProc("hid_init")
		openProc := dll.NewProc("hid_open")
		writeProc := dll.NewProc("hid_write")
		getInputProc := dll.NewProc("hid_get_input_report")
		closeProc := dll.NewProc("hid_close")
		exitProc := dll.NewProc("hid_exit")
		enumerateProc := dll.NewProc("hid_enumerate")
		freeEnumProc := dll.NewProc("hid_free_enumeration")

		if err := dll.Load(); err != nil {
			lastErr = err
			continue
		}
		if err := initProc.Find(); err != nil {
			lastErr = err
			continue
		}
		if err := openProc.Find(); err != nil {
			lastErr = err
			continue
		}
		if err := writeProc.Find(); err != nil {
			lastErr = err
			continue
		}
		// hid_get_input_report is optional across hidapi builds; use it when present.
		if err := getInputProc.Find(); err != nil {
			getInputProc = nil
		}
		// hid_enumerate and hid_free_enumeration are optional; present in standard hidapi builds.
		if err := enumerateProc.Find(); err != nil {
			enumerateProc = nil
		}
		if err := freeEnumProc.Find(); err != nil {
			freeEnumProc = nil
		}
		if err := closeProc.Find(); err != nil {
			lastErr = err
			continue
		}
		if err := exitProc.Find(); err != nil {
			lastErr = err
			continue
		}

		bridge := &hidBridge{
			dll:           dll,
			initProc:      initProc,
			openProc:      openProc,
			writeProc:     writeProc,
			getInputProc:  getInputProc,
			closeProc:     closeProc,
			exitProc:      exitProc,
			enumerateProc: enumerateProc,
			freeEnumProc:  freeEnumProc,
		}

		if _, _, callErr := bridge.initProc.Call(); callErr != syscall.Errno(0) {
			lastErr = callErr
			continue
		}

		return bridge, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unable to load hidapi.dll")
	}
	return nil, fmt.Errorf("hidapi unavailable: %w", lastErr)
}

func (b *hidBridge) open(vendorID uint16, productID uint16) (uintptr, error) {
	handle, _, callErr := b.openProc.Call(uintptr(vendorID), uintptr(productID), 0)
	if handle == 0 {
		if callErr == syscall.Errno(0) {
			return 0, fmt.Errorf("hid_open returned null handle")
		}
		return 0, fmt.Errorf("hid_open failed: %w", callErr)
	}
	return handle, nil
}

func (b *hidBridge) write(handle uintptr, payload []byte) error {
	report := make([]byte, 65)
	copy(report, payload)

	written, _, callErr := b.writeProc.Call(handle, uintptr(unsafe.Pointer(&report[0])), uintptr(len(report)))
	if int(written) <= 0 {
		if callErr == syscall.Errno(0) {
			return fmt.Errorf("hid_write returned %d", int(written))
		}
		return fmt.Errorf("hid_write failed: %w", callErr)
	}

	time.Sleep(5 * time.Millisecond)
	return nil
}

func (b *hidBridge) writeRaw(handle uintptr, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("hid_write payload is empty")
	}

	written, _, callErr := b.writeProc.Call(handle, uintptr(unsafe.Pointer(&payload[0])), uintptr(len(payload)))
	if int(written) <= 0 {
		if callErr == syscall.Errno(0) {
			return fmt.Errorf("hid_write returned %d", int(written))
		}
		return fmt.Errorf("hid_write failed: %w", callErr)
	}

	time.Sleep(5 * time.Millisecond)
	return nil
}

func (b *hidBridge) sendStartAction(handle uintptr, channel int, numFans byte) error {
	report := make([]byte, 65)
	report[0x00] = 0xE0
	report[0x01] = 0x10
	report[0x02] = 0x60
	report[0x03] = byte(1 + (channel / 2))
	report[0x04] = numFans
	return b.writeRaw(handle, report)
}

func (b *hidBridge) sendColorData(handle uintptr, channel int, numLEDs int, ledData []byte) error {
	report := make([]byte, 353)
	report[0x00] = 0xE0
	report[0x01] = byte(0x30 + channel)

	maxData := numLEDs * 3
	if maxData > len(ledData) {
		maxData = len(ledData)
	}
	if maxData > len(report)-2 {
		maxData = len(report) - 2
	}
	copy(report[2:], ledData[:maxData])

	return b.writeRaw(handle, report)
}

func (b *hidBridge) sendCommitAction(handle uintptr, channel int, effect byte, speed byte, direction byte, brightness byte) error {
	report := make([]byte, 65)
	report[0x00] = 0xE0
	report[0x01] = byte(0x10 + channel)
	report[0x02] = effect
	report[0x03] = speed
	report[0x04] = direction
	report[0x05] = brightness
	return b.writeRaw(handle, report)
}

func (b *hidBridge) getInputReport(handle uintptr, reportID byte) ([]byte, error) {
	if b.getInputProc == nil {
		return nil, fmt.Errorf("hid_get_input_report is not available in current hidapi.dll")
	}

	report := make([]byte, 65)
	report[0] = reportID

	read, _, callErr := b.getInputProc.Call(handle, uintptr(unsafe.Pointer(&report[0])), uintptr(len(report)))
	if int(read) <= 0 {
		if callErr == syscall.Errno(0) {
			return nil, fmt.Errorf("hid_get_input_report returned %d", int(read))
		}
		return nil, fmt.Errorf("hid_get_input_report failed: %w", callErr)
	}

	return report, nil
}

func (b *hidBridge) close(handle uintptr) {
	if handle != 0 {
		b.closeProc.Call(handle)
	}
	b.exitProc.Call()
}

func hidBrightnessCode(brightnessPct int) byte {
	switch {
	case brightnessPct <= 0:
		return 0x08
	case brightnessPct <= 25:
		return 0x03
	case brightnessPct <= 50:
		return 0x02
	case brightnessPct <= 75:
		return 0x01
	default:
		return 0x00
	}
}

func buildStaticColorLEDData(red uint8, green uint8, blue uint8, brightnessPct int) []byte {
	const numLEDs = 80
	data := make([]byte, numLEDs*3)

	sum := int(red) + int(green) + int(blue)
	limitScale := 1.0
	if sum > 460 {
		limitScale = 460.0 / float64(sum)
	}
	brightnessScale := float64(brightnessPct) / 100.0
	scale := limitScale * brightnessScale

	scaledR := uint8(float64(red) * scale)
	scaledB := uint8(float64(blue) * scale)
	scaledG := uint8(float64(green) * scale)

	for i := 0; i < numLEDs; i++ {
		off := i * 3
		// SL Infinity LED payload uses RBG ordering.
		data[off+0] = scaledR
		data[off+1] = scaledB
		data[off+2] = scaledG
	}

	return data
}

func buildPaletteLEDData(colors []effectColor, brightnessPct int) []byte {
	if len(colors) == 0 {
		return make([]byte, 80*3)
	}

	const numLEDs = 80
	const ledsPerFan = 16
	data := make([]byte, numLEDs*3)

	for led := 0; led < numLEDs; led++ {
		fanIdx := led / ledsPerFan
		color := colors[fanIdx%len(colors)]
		r, b, g := scaleEffectColor(color, brightnessPct)
		off := led * 3
		data[off+0] = r
		data[off+1] = b
		data[off+2] = g
	}

	return data
}

func scaleEffectColor(color effectColor, brightnessPct int) (uint8, uint8, uint8) {
	sum := int(color.R) + int(color.G) + int(color.B)
	limitScale := 1.0
	if sum > 460 {
		limitScale = 460.0 / float64(sum)
	}
	brightnessScale := float64(brightnessPct) / 100.0
	scale := limitScale * brightnessScale

	return uint8(float64(color.R) * scale), uint8(float64(color.B) * scale), uint8(float64(color.G) * scale)
}
