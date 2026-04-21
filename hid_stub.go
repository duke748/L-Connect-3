//go:build !windows

package main

import "fmt"

func hidProbe(vendorID uint16, productID uint16) (string, error) {
	return "", fmt.Errorf("direct HID fallback is only implemented on Windows")
}

func hidFanSet(vendorID uint16, productID uint16, port int, speed int) error {
	return fmt.Errorf("direct HID fallback is only implemented on Windows")
}

func hidSetStaticColorAll(vendorID uint16, productID uint16, red uint8, green uint8, blue uint8, brightnessPct int) error {
	return fmt.Errorf("direct HID fallback is only implemented on Windows")
}

func hidSetStaticColorPort(vendorID uint16, productID uint16, port int, red uint8, green uint8, blue uint8, brightnessPct int) error {
	return fmt.Errorf("direct HID fallback is only implemented on Windows")
}

func hidSetStaticColorChannelByID(vendorID uint16, productID uint16, channel int, red uint8, green uint8, blue uint8, brightnessPct int) error {
	return fmt.Errorf("direct HID fallback is only implemented on Windows")
}

func hidApplyEffectChannelByID(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, hasColor bool, red uint8, green uint8, blue uint8) error {
	return fmt.Errorf("direct HID fallback is only implemented on Windows")
}

func hidApplyEffectPaletteChannelByID(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, colors []effectColor) error {
	return fmt.Errorf("direct HID fallback is only implemented on Windows")
}

func hidReadRPM(vendorID uint16, productID uint16) ([4]uint16, error) {
	return [4]uint16{}, fmt.Errorf("direct HID fallback is only implemented on Windows")
}
