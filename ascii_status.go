package main

import (
	"fmt"
	"strings"
)

type asciiPortView struct {
	number   int
	channels [2]int
	light    lightingPortState
}

func runASCIIStatus() error {
	channelMap, err := loadHIDPortChannelMap()
	if err != nil {
		return err
	}

	lightingState, err := loadLightingState()
	if err != nil {
		return err
	}

	fanSnapshot, fanErr := loadFanSnapshot()
	rpmByPort, rpmErr := hidReadRPM(slInfinityVID, slInfinityPID)

	ports := []asciiPortView{
		{number: 1, channels: channelMap.Port1, light: lightingState.Ports["port1"]},
		{number: 2, channels: channelMap.Port2, light: lightingState.Ports["port2"]},
		{number: 3, channels: channelMap.Port3, light: lightingState.Ports["port3"]},
		{number: 4, channels: channelMap.Port4, light: lightingState.Ports["port4"]},
	}

	leftTop := renderPortASCII(ports[0], fanSnapshot, fanErr, rpmByPort, rpmErr)
	rightTop := renderPortASCII(ports[1], fanSnapshot, fanErr, rpmByPort, rpmErr)
	leftBottom := renderPortASCII(ports[2], fanSnapshot, fanErr, rpmByPort, rpmErr)
	rightBottom := renderPortASCII(ports[3], fanSnapshot, fanErr, rpmByPort, rpmErr)

	fmt.Println("l-connect3-cli ascii status")
	fmt.Println()
	printASCIIBoxRow(leftTop, rightTop)
	printASCIIBoxRow(leftBottom, rightBottom)
	fmt.Println()
	fmt.Println("Legend: o = fan, [] = port bay, values come from last CLI-applied state plus live RPM when available")
	if fanErr != nil {
		fmt.Printf("Fan target state unavailable: %v\n", fanErr)
	}
	if rpmErr != nil {
		fmt.Printf("RPM unavailable: %v\n", rpmErr)
	}

	return nil
}

func printASCIIBoxRow(left []string, right []string) {
	fmt.Println("+----------------------+  +----------------------+")
	for i := 0; i < len(left); i++ {
		fmt.Printf("| %-20s |  | %-20s |\n", left[i], right[i])
	}
	fmt.Println("+----------------------+  +----------------------+")
}

func renderPortASCII(port asciiPortView, fanSnapshot fanPortSnapshot, fanErr error, rpmByPort [4]uint16, rpmErr error) []string {
	return []string{
		fmt.Sprintf("P%d [o-o-o-o]", port.number),
		fmt.Sprintf("CH %d,%d", port.channels[0], port.channels[1]),
		renderEffectLine(port.light),
		renderColorLine(port.light),
		renderFanLine(port.number, fanSnapshot, fanErr, rpmByPort, rpmErr),
	}
}

func renderEffectLine(light lightingPortState) string {
	if light.Layout == "split" && light.SecondaryMode != "" {
		return compactLabel(fmt.Sprintf("FX %s+%s", light.Mode, light.SecondaryMode), 20)
	}
	if light.Layout == "linked" {
		return compactLabel(fmt.Sprintf("FX %s [L]", light.Mode), 20)
	}
	return compactLabel(fmt.Sprintf("FX %s", light.Mode), 20)
}

func renderColorLine(light lightingPortState) string {
	if light.Layout == "split" {
		return compactLabel(fmt.Sprintf("C A:%d B:%d", len(light.Colors), len(light.SecondaryColors)), 20)
	}
	if light.Layout == "linked" {
		return compactLabel(fmt.Sprintf("C linked:%d", len(light.Colors)), 20)
	}
	if light.Color != "" {
		return compactLabel(fmt.Sprintf("CL #%s %d%%", light.Color, light.Brightness), 20)
	}
	if light.Brightness > 0 {
		return compactLabel(fmt.Sprintf("BR %d%% SPD %d", light.Brightness, light.Speed), 20)
	}
	return "CL unknown"
}

func renderFanLine(portNumber int, fanSnapshot fanPortSnapshot, fanErr error, rpmByPort [4]uint16, rpmErr error) string {
	fanLabel := "fan ?"
	if fanErr == nil {
		state := fanSnapshot.Ports[fmt.Sprintf("port%d", portNumber)]
		if state.Mode == "preset" {
			fanLabel = fmt.Sprintf("fan %s", compactLabel(state.Preset, 8))
		} else if state.Mode == "manual" {
			fanLabel = fmt.Sprintf("fan %d%%", state.Speed)
		} else {
			fanLabel = "fan unknown"
		}
	}

	rpmLabel := "rpm ?"
	if rpmErr == nil && portNumber >= 1 && portNumber <= len(rpmByPort) {
		rpmLabel = fmt.Sprintf("rpm %d", rpmByPort[portNumber-1])
	}

	return compactLabel(fanLabel+" "+rpmLabel, 20)
}

func compactLabel(input string, width int) string {
	clean := strings.TrimSpace(input)
	if len(clean) <= width {
		return clean
	}
	if width <= 3 {
		return clean[:width]
	}
	return clean[:width-3] + "..."
}
