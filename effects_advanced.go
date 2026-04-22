package main

import (
	"fmt"
	"strings"
	"time"
)

type effectColor struct {
	R uint8
	G uint8
	B uint8
	H string
}

func runHIDEffectLinked(effectArg string, colorsArg string, port int, speed int, brightnessPct int, direction int) error {
	effectCode, effectLabel, err := parseEffectCode(effectArg)
	if err != nil {
		return err
	}

	if port < 0 || port > 4 {
		return fmt.Errorf("port must be 0..4 (0 means all mapped ports)")
	}
	if speed < 0 || speed > 255 {
		return fmt.Errorf("speed must be 0..255")
	}
	if brightnessPct < 0 || brightnessPct > 100 {
		return fmt.Errorf("brightness must be 0..100")
	}
	if direction < 0 || direction > 255 {
		return fmt.Errorf("direction must be 0..255")
	}

	colors, cleanHexes, err := parseHexColorList(colorsArg, 4)
	if err != nil {
		return err
	}

	channelMap, err := loadHIDPortChannelMap()
	if err != nil {
		return err
	}

	targetPorts := targetPortsForEffect(port)
	for _, p := range targetPorts {
		ch := channelsForPort(channelMap, p)
		for _, channel := range ch {
			if err := hidApplyEffectPaletteChannelByIDFn(slInfinityVID, slInfinityPID, channel, effectCode, speed, direction, brightnessPct, colors); err != nil {
				return fmt.Errorf("apply linked effect failed for port %d channel %d: %w", p, channel, err)
			}
		}

		state := lightingPortState{
			Mode:       effectLabel,
			EffectCode: fmt.Sprintf("0x%02X", effectCode),
			Layout:     "linked",
			Colors:     cleanHexes,
			Brightness: brightnessPct,
			Speed:      speed,
			Direction:  direction,
			UpdatedAt:  time.Now().Format(time.RFC3339),
		}
		if len(cleanHexes) > 0 {
			state.Color = cleanHexes[0]
		}
		_ = saveLightingStateForPort(p, "hid-effect-linked", state)
	}

	if port == 0 {
		fmt.Printf("hid linked effect applied: mode=%s(0x%02X) ports=1-4 colors=%s speed=%d brightness=%d%% direction=%d\n", effectLabel, effectCode, strings.Join(cleanHexes, ","), speed, brightnessPct, direction)
		return nil
	}
	fmt.Printf("hid linked effect applied: port=%d mode=%s(0x%02X) colors=%s speed=%d brightness=%d%% direction=%d\n", port, effectLabel, effectCode, strings.Join(cleanHexes, ","), speed, brightnessPct, direction)
	return nil
}

func runHIDEffectSplit(primaryArg string, secondaryArg string, primaryColorsArg string, secondaryColorsArg string, port int, speed int, brightnessPct int, direction int) error {
	primaryCode, primaryLabel, err := parseEffectCode(primaryArg)
	if err != nil {
		return err
	}
	secondaryCode, secondaryLabel, err := parseEffectCode(secondaryArg)
	if err != nil {
		return err
	}

	if port < 0 || port > 4 {
		return fmt.Errorf("port must be 0..4 (0 means all mapped ports)")
	}
	if speed < 0 || speed > 255 {
		return fmt.Errorf("speed must be 0..255")
	}
	if brightnessPct < 0 || brightnessPct > 100 {
		return fmt.Errorf("brightness must be 0..100")
	}
	if direction < 0 || direction > 255 {
		return fmt.Errorf("direction must be 0..255")
	}

	primaryColors, primaryHexes, err := parseHexColorList(primaryColorsArg, 4)
	if err != nil {
		return fmt.Errorf("primary-colors: %w", err)
	}
	secondaryColors, secondaryHexes, err := parseHexColorList(secondaryColorsArg, 4)
	if err != nil {
		return fmt.Errorf("secondary-colors: %w", err)
	}

	channelMap, err := loadHIDPortChannelMap()
	if err != nil {
		return err
	}

	targetPorts := targetPortsForEffect(port)
	for _, p := range targetPorts {
		ch := channelsForPort(channelMap, p)

		if err := hidApplyEffectPaletteChannelByIDFn(slInfinityVID, slInfinityPID, ch[0], primaryCode, speed, direction, brightnessPct, primaryColors); err != nil {
			return fmt.Errorf("apply split primary failed for port %d channel %d: %w", p, ch[0], err)
		}
		if err := hidApplyEffectPaletteChannelByIDFn(slInfinityVID, slInfinityPID, ch[1], secondaryCode, speed, direction, brightnessPct, secondaryColors); err != nil {
			return fmt.Errorf("apply split secondary failed for port %d channel %d: %w", p, ch[1], err)
		}

		state := lightingPortState{
			Mode:                primaryLabel,
			EffectCode:          fmt.Sprintf("0x%02X", primaryCode),
			Layout:              "split",
			Colors:              primaryHexes,
			SecondaryMode:       secondaryLabel,
			SecondaryEffectCode: fmt.Sprintf("0x%02X", secondaryCode),
			SecondaryColors:     secondaryHexes,
			Brightness:          brightnessPct,
			Speed:               speed,
			Direction:           direction,
			UpdatedAt:           time.Now().Format(time.RFC3339),
		}
		if len(primaryHexes) > 0 {
			state.Color = primaryHexes[0]
		}
		_ = saveLightingStateForPort(p, "hid-effect-split", state)
	}

	if port == 0 {
		fmt.Printf("hid split effect applied: ports=1-4 primary=%s(0x%02X) secondary=%s(0x%02X)\n", primaryLabel, primaryCode, secondaryLabel, secondaryCode)
		return nil
	}
	fmt.Printf("hid split effect applied: port=%d primary=%s(0x%02X) secondary=%s(0x%02X)\n", port, primaryLabel, primaryCode, secondaryLabel, secondaryCode)
	return nil
}

func parseHexColorList(input string, maxColors int) ([]effectColor, []string, error) {
	parts := make([]string, 0)
	for _, raw := range strings.Split(input, ",") {
		clean := strings.TrimSpace(raw)
		if clean != "" {
			parts = append(parts, clean)
		}
	}

	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("at least one color is required")
	}
	if len(parts) > maxColors {
		return nil, nil, fmt.Errorf("at most %d colors are allowed", maxColors)
	}

	colors := make([]effectColor, 0, len(parts))
	hexes := make([]string, 0, len(parts))
	for _, p := range parts {
		h, r, g, b, err := parseHexColor(p)
		if err != nil {
			return nil, nil, err
		}
		colors = append(colors, effectColor{R: r, G: g, B: b, H: h})
		hexes = append(hexes, h)
	}

	return colors, hexes, nil
}

func targetPortsForEffect(port int) []int {
	if port == 0 {
		return []int{1, 2, 3, 4}
	}
	return []int{port}
}
