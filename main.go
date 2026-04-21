// Command l-connect3-cli controls a Lian Li SL Infinity hub via direct HID.
//
// It is built for scripting and automation without requiring the full L-Connect UI.
// The current command set covers fan speed, static color, telemetry, and
// persisted channel mapping.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	slInfinityVID = 0x0CF2
	slInfinityPID = 0xA102
	stateFilePath = ".l-connect3-cli-state.json"
	mapFilePath   = ".l-connect3-cli-map.json"
	lightFilePath = ".l-connect3-cli-lighting.json"
	fanFilePath   = ".l-connect3-cli-fans.json"
)

type fanTargetState struct {
	Source    string `json:"Source"`
	Mode      string `json:"Mode"`
	Speed     int    `json:"Speed"`
	Preset    string `json:"Preset"`
	UpdatedAt string `json:"UpdatedAt"`
}

type hidPortChannelMap struct {
	Port1 [2]int `json:"Port1"`
	Port2 [2]int `json:"Port2"`
	Port3 [2]int `json:"Port3"`
	Port4 [2]int `json:"Port4"`
}

type lightingPortState struct {
	Mode                string   `json:"Mode"`
	EffectCode          string   `json:"EffectCode,omitempty"`
	Layout              string   `json:"Layout,omitempty"`
	Color               string   `json:"Color,omitempty"`
	Colors              []string `json:"Colors,omitempty"`
	SecondaryMode       string   `json:"SecondaryMode,omitempty"`
	SecondaryEffectCode string   `json:"SecondaryEffectCode,omitempty"`
	SecondaryColors     []string `json:"SecondaryColors,omitempty"`
	Brightness          int      `json:"Brightness"`
	Speed               int      `json:"Speed,omitempty"`
	Direction           int      `json:"Direction,omitempty"`
	UpdatedAt           string   `json:"UpdatedAt"`
}

type lightingState struct {
	Source string                       `json:"Source"`
	Ports  map[string]lightingPortState `json:"Ports"`
}

type fanPortState struct {
	Mode      string `json:"Mode"`
	Speed     int    `json:"Speed"`
	Preset    string `json:"Preset,omitempty"`
	UpdatedAt string `json:"UpdatedAt"`
}

type fanPortSnapshot struct {
	Source string                  `json:"Source"`
	Ports  map[string]fanPortState `json:"Ports"`
}

// main dispatches HID-only control commands through Cobra.
func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runHIDProbe() error {
	result, err := hidProbe(slInfinityVID, slInfinityPID)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func runHIDFan(portArg string, speedArg string) error {
	port, err := strconv.Atoi(portArg)
	if err != nil || port < 1 || port > 4 {
		return fmt.Errorf("port must be 1..4")
	}

	speed, err := strconv.Atoi(speedArg)
	if err != nil || speed < 0 || speed > 100 {
		return fmt.Errorf("speed must be 0..100")
	}

	if err := hidFanSet(slInfinityVID, slInfinityPID, port, speed); err != nil {
		return err
	}

	_ = saveFanTargetState(fanTargetState{
		Source:    "hid-fan",
		Mode:      "manual",
		Speed:     speed,
		Preset:    "",
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	_ = saveFanSnapshotForPort(port, "hid-fan", fanPortState{
		Mode:      "manual",
		Speed:     speed,
		UpdatedAt: time.Now().Format(time.RFC3339),
	})

	fmt.Printf("hid fan command sent: port=%d speed=%d%%\n", port, speed)
	return nil
}

func runHIDSet(hexColor string, brightnessArg string) error {
	brightnessPct, err := strconv.Atoi(brightnessArg)
	if err != nil || brightnessPct < 0 || brightnessPct > 100 {
		return fmt.Errorf("hid-set brightness must be 0..100")
	}

	cleanHex, red, green, blue, err := parseHexColor(hexColor)
	if err != nil {
		return err
	}

	if err := hidSetStaticColorAll(slInfinityVID, slInfinityPID, red, green, blue, brightnessPct); err != nil {
		return err
	}

	_ = saveLightingStateForAllPorts("hid-set", lightingPortState{
		Mode:       "static",
		EffectCode: "0x01",
		Layout:     "single",
		Color:      cleanHex,
		Colors:     []string{cleanHex},
		Brightness: brightnessPct,
		UpdatedAt:  time.Now().Format(time.RFC3339),
	})

	fmt.Printf("hid color set: #%s brightness=%d%% on channels 1-4\n", cleanHex, brightnessPct)
	return nil
}

func runHIDSetPort(portArg string, hexColor string, brightnessArg string) error {
	port, err := strconv.Atoi(portArg)
	if err != nil || port < 1 || port > 4 {
		return fmt.Errorf("port must be 1..4")
	}

	brightnessPct, err := strconv.Atoi(brightnessArg)
	if err != nil || brightnessPct < 0 || brightnessPct > 100 {
		return fmt.Errorf("hid-set-port brightness must be 0..100")
	}

	cleanHex, red, green, blue, err := parseHexColor(hexColor)
	if err != nil {
		return err
	}

	channelMap, err := loadHIDPortChannelMap()
	if err != nil {
		return err
	}

	channels := channelsForPort(channelMap, port)
	for _, channel := range channels {
		if err := hidSetStaticColorChannelByID(slInfinityVID, slInfinityPID, channel, red, green, blue, brightnessPct); err != nil {
			return err
		}
	}

	_ = saveLightingStateForPort(port, "hid-set-port", lightingPortState{
		Mode:       "static",
		EffectCode: "0x01",
		Layout:     "single",
		Color:      cleanHex,
		Colors:     []string{cleanHex},
		Brightness: brightnessPct,
		UpdatedAt:  time.Now().Format(time.RFC3339),
	})

	fmt.Printf("hid color set: port=%d channels=%d,%d #%s brightness=%d%%\n", port, channels[0], channels[1], cleanHex, brightnessPct)
	return nil
}

func runHIDSetChannel(channelArg string, hexColor string, brightnessArg string) error {
	channel, err := strconv.Atoi(channelArg)
	if err != nil || channel < 0 || channel > 7 {
		return fmt.Errorf("channel must be 0..7")
	}

	brightnessPct, err := strconv.Atoi(brightnessArg)
	if err != nil || brightnessPct < 0 || brightnessPct > 100 {
		return fmt.Errorf("hid-set-channel brightness must be 0..100")
	}

	cleanHex, red, green, blue, err := parseHexColor(hexColor)
	if err != nil {
		return err
	}

	if err := hidSetStaticColorChannelByID(slInfinityVID, slInfinityPID, channel, red, green, blue, brightnessPct); err != nil {
		return err
	}

	portLabel := rawChannelPortLabel(channel)
	if portLabel != "" {
		portNumber := int(portLabel[len(portLabel)-1] - '0')
		_ = saveLightingStateForPort(portNumber, "hid-set-channel", lightingPortState{
			Mode:       "static",
			EffectCode: "0x01",
			Layout:     "single",
			Color:      cleanHex,
			Colors:     []string{cleanHex},
			Brightness: brightnessPct,
			UpdatedAt:  time.Now().Format(time.RFC3339),
		})
	}

	fmt.Printf("hid color set: channel=%d #%s brightness=%d%%\n", channel, cleanHex, brightnessPct)
	return nil
}

func runHIDMapShow() error {
	channelMap, err := loadHIDPortChannelMap()
	if err != nil {
		return err
	}

	fmt.Println("hid port map")
	fmt.Printf("port 1 -> channels %d,%d\n", channelMap.Port1[0], channelMap.Port1[1])
	fmt.Printf("port 2 -> channels %d,%d\n", channelMap.Port2[0], channelMap.Port2[1])
	fmt.Printf("port 3 -> channels %d,%d\n", channelMap.Port3[0], channelMap.Port3[1])
	fmt.Printf("port 4 -> channels %d,%d\n", channelMap.Port4[0], channelMap.Port4[1])
	return nil
}

func runHIDMapSet(portArg string, channelAArg string, channelBArg string) error {
	port, err := strconv.Atoi(portArg)
	if err != nil || port < 1 || port > 4 {
		return fmt.Errorf("port must be 1..4")
	}

	channelA, err := strconv.Atoi(channelAArg)
	if err != nil || channelA < 0 || channelA > 7 {
		return fmt.Errorf("channelA must be 0..7")
	}

	channelB, err := strconv.Atoi(channelBArg)
	if err != nil || channelB < 0 || channelB > 7 {
		return fmt.Errorf("channelB must be 0..7")
	}

	if channelA == channelB {
		return fmt.Errorf("channelA and channelB must be different")
	}

	channelMap, err := loadHIDPortChannelMap()
	if err != nil {
		return err
	}

	switch port {
	case 1:
		channelMap.Port1 = [2]int{channelA, channelB}
	case 2:
		channelMap.Port2 = [2]int{channelA, channelB}
	case 3:
		channelMap.Port3 = [2]int{channelA, channelB}
	case 4:
		channelMap.Port4 = [2]int{channelA, channelB}
	}

	if err := saveHIDPortChannelMap(channelMap); err != nil {
		return err
	}

	fmt.Printf("hid map updated: port %d -> channels %d,%d\n", port, channelA, channelB)
	fmt.Printf("note: mapping only; run hid-set-port %d <hex-color> [brightness] to apply a visible change\n", port)
	return nil
}

func runFanAll(speedOrPresetArg string) error {
	speed, presetLabel, err := parseFanSpeedOrPreset(speedOrPresetArg)
	if err != nil {
		return err
	}

	failedPorts := make([]int, 0, 4)
	for port := 1; port <= 4; port++ {
		if err := hidFanSet(slInfinityVID, slInfinityPID, port, speed); err != nil {
			failedPorts = append(failedPorts, port)
		}
	}

	if len(failedPorts) > 0 {
		return fmt.Errorf("failed to apply fan speed to ports: %v", failedPorts)
	}

	if presetLabel != "" {
		_ = saveFanTargetState(fanTargetState{
			Source:    "fan-all",
			Mode:      "preset",
			Speed:     speed,
			Preset:    presetLabel,
			UpdatedAt: time.Now().Format(time.RFC3339),
		})
		_ = saveFanSnapshotForAllPorts("fan-all", fanPortState{
			Mode:      "preset",
			Speed:     speed,
			Preset:    presetLabel,
			UpdatedAt: time.Now().Format(time.RFC3339),
		})
		fmt.Printf("fan-all applied preset %q (%d%%) to ports 1-4\n", presetLabel, speed)
		return nil
	}

	_ = saveFanTargetState(fanTargetState{
		Source:    "fan-all",
		Mode:      "manual",
		Speed:     speed,
		Preset:    "",
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
	_ = saveFanSnapshotForAllPorts("fan-all", fanPortState{
		Mode:      "manual",
		Speed:     speed,
		UpdatedAt: time.Now().Format(time.RFC3339),
	})

	fmt.Printf("fan-all applied %d%% to ports 1-4\n", speed)
	return nil
}

func runHIDRPM() error {
	rpmByPort, err := hidReadRPM(slInfinityVID, slInfinityPID)
	if err != nil {
		return err
	}

	fmt.Println("current fan rpm (hid):")
	for i := 0; i < len(rpmByPort); i++ {
		fmt.Printf("port %d: %d RPM\n", i+1, rpmByPort[i])
	}

	return nil
}

func runHIDStatus() error {
	rpmByPort, err := hidReadRPM(slInfinityVID, slInfinityPID)
	if err != nil {
		return err
	}

	fmt.Println("hid status")
	fmt.Println("rpm:")
	for i := 0; i < len(rpmByPort); i++ {
		fmt.Printf("  port %d: %d RPM\n", i+1, rpmByPort[i])
	}

	state, err := loadFanTargetState()
	if err != nil {
		fmt.Printf("last target: unavailable (%v)\n", err)
		return nil
	}

	if state.Mode == "preset" {
		fmt.Printf("last target: %s preset=%q speed=%d%% at %s\n", state.Source, state.Preset, state.Speed, state.UpdatedAt)
		return nil
	}

	fmt.Printf("last target: %s speed=%d%% at %s\n", state.Source, state.Speed, state.UpdatedAt)
	return nil
}

func runHIDEffect(effectArg string, colorArg string, port int, speed int, brightnessPct int, direction int) error {
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

	hasColor := strings.TrimSpace(colorArg) != ""
	var red, green, blue uint8
	cleanHex := ""
	if hasColor {
		var parseErr error
		cleanHex, red, green, blue, parseErr = parseHexColor(colorArg)
		if parseErr != nil {
			return parseErr
		}
	}

	if port == 0 {
		appliedAt := time.Now().Format(time.RFC3339)
		for channel := 0; channel < 8; channel += 2 {
			if err := hidApplyEffectChannelByID(slInfinityVID, slInfinityPID, channel, effectCode, speed, direction, brightnessPct, hasColor, red, green, blue); err != nil {
				return fmt.Errorf("apply effect failed for channel %d: %w", channel, err)
			}
		}
		portState := lightingPortState{
			Mode:       effectLabel,
			EffectCode: fmt.Sprintf("0x%02X", effectCode),
			Layout:     "single",
			Brightness: brightnessPct,
			Speed:      speed,
			Direction:  direction,
			UpdatedAt:  appliedAt,
		}
		if hasColor {
			portState.Color = cleanHex
			portState.Colors = []string{cleanHex}
		}
		_ = saveLightingStateForAllPorts("hid-effect", portState)
		if hasColor {
			fmt.Printf("hid effect applied: mode=%s(0x%02X) channels=0,2,4,6 color=#%s speed=%d brightness=%d%% direction=%d\n", effectLabel, effectCode, cleanHex, speed, brightnessPct, direction)
			return nil
		}
		fmt.Printf("hid effect applied: mode=%s(0x%02X) channels=0,2,4,6 speed=%d brightness=%d%% direction=%d\n", effectLabel, effectCode, speed, brightnessPct, direction)
		return nil
	}

	channelMap, err := loadHIDPortChannelMap()
	if err != nil {
		return err
	}

	channels := channelsForPort(channelMap, port)
	for _, channel := range channels {
		if err := hidApplyEffectChannelByID(slInfinityVID, slInfinityPID, channel, effectCode, speed, direction, brightnessPct, hasColor, red, green, blue); err != nil {
			return fmt.Errorf("apply effect failed for channel %d: %w", channel, err)
		}
	}

	portState := lightingPortState{
		Mode:       effectLabel,
		EffectCode: fmt.Sprintf("0x%02X", effectCode),
		Layout:     "single",
		Brightness: brightnessPct,
		Speed:      speed,
		Direction:  direction,
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}
	if hasColor {
		portState.Color = cleanHex
		portState.Colors = []string{cleanHex}
	}
	_ = saveLightingStateForPort(port, "hid-effect", portState)

	if hasColor {
		fmt.Printf("hid effect applied: port=%d channels=%d,%d mode=%s(0x%02X) color=#%s speed=%d brightness=%d%% direction=%d\n", port, channels[0], channels[1], effectLabel, effectCode, cleanHex, speed, brightnessPct, direction)
		return nil
	}
	fmt.Printf("hid effect applied: port=%d channels=%d,%d mode=%s(0x%02X) speed=%d brightness=%d%% direction=%d\n", port, channels[0], channels[1], effectLabel, effectCode, speed, brightnessPct, direction)
	return nil
}

func saveFanTargetState(state fanTargetState) error {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFilePath, payload, 0o644)
}

func loadFanTargetState() (fanTargetState, error) {
	var state fanTargetState

	payload, err := os.ReadFile(stateFilePath)
	if err != nil {
		return state, err
	}

	if err := json.Unmarshal(payload, &state); err != nil {
		return state, err
	}

	return state, nil
}

func defaultHIDPortChannelMap() hidPortChannelMap {
	return hidPortChannelMap{
		Port1: [2]int{0, 1},
		Port2: [2]int{2, 3},
		Port3: [2]int{4, 5},
		Port4: [2]int{6, 7},
	}
}

func loadHIDPortChannelMap() (hidPortChannelMap, error) {
	channelMap := defaultHIDPortChannelMap()

	payload, err := os.ReadFile(mapFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return channelMap, nil
		}
		return channelMap, err
	}

	if err := json.Unmarshal(payload, &channelMap); err != nil {
		return channelMap, err
	}

	if err := validateHIDPortChannelMap(channelMap); err != nil {
		return channelMap, err
	}

	return channelMap, nil
}

func saveHIDPortChannelMap(channelMap hidPortChannelMap) error {
	if err := validateHIDPortChannelMap(channelMap); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(channelMap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(mapFilePath, payload, 0o644)
}

func defaultLightingState() lightingState {
	updatedAt := time.Now().Format(time.RFC3339)
	return lightingState{
		Source: "default",
		Ports: map[string]lightingPortState{
			"port1": {Mode: "unknown", UpdatedAt: updatedAt},
			"port2": {Mode: "unknown", UpdatedAt: updatedAt},
			"port3": {Mode: "unknown", UpdatedAt: updatedAt},
			"port4": {Mode: "unknown", UpdatedAt: updatedAt},
		},
	}
}

func loadLightingState() (lightingState, error) {
	state := defaultLightingState()

	payload, err := os.ReadFile(lightFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}

	if err := json.Unmarshal(payload, &state); err != nil {
		return state, err
	}

	if state.Ports == nil {
		state.Ports = defaultLightingState().Ports
	}

	for _, portKey := range []string{"port1", "port2", "port3", "port4"} {
		if _, ok := state.Ports[portKey]; !ok {
			state.Ports[portKey] = lightingPortState{Mode: "unknown", UpdatedAt: time.Now().Format(time.RFC3339)}
		}
	}

	return state, nil
}

func saveLightingState(state lightingState) error {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(lightFilePath, payload, 0o644)
}

func saveLightingStateForAllPorts(source string, portState lightingPortState) error {
	state, err := loadLightingState()
	if err != nil {
		return err
	}

	state.Source = source
	for _, portKey := range []string{"port1", "port2", "port3", "port4"} {
		state.Ports[portKey] = portState
	}

	return saveLightingState(state)
}

func saveLightingStateForPort(port int, source string, portState lightingPortState) error {
	state, err := loadLightingState()
	if err != nil {
		return err
	}

	state.Source = source
	state.Ports[fmt.Sprintf("port%d", port)] = portState
	return saveLightingState(state)
}

func rawChannelPortLabel(channel int) string {
	switch channel {
	case 0, 1:
		return "port1"
	case 2, 3:
		return "port2"
	case 4, 5:
		return "port3"
	case 6, 7:
		return "port4"
	default:
		return ""
	}
}

func defaultFanSnapshot() fanPortSnapshot {
	updatedAt := time.Now().Format(time.RFC3339)
	return fanPortSnapshot{
		Source: "default",
		Ports: map[string]fanPortState{
			"port1": {Mode: "unknown", UpdatedAt: updatedAt},
			"port2": {Mode: "unknown", UpdatedAt: updatedAt},
			"port3": {Mode: "unknown", UpdatedAt: updatedAt},
			"port4": {Mode: "unknown", UpdatedAt: updatedAt},
		},
	}
}

func loadFanSnapshot() (fanPortSnapshot, error) {
	snapshot := defaultFanSnapshot()

	payload, err := os.ReadFile(fanFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot, nil
		}
		return snapshot, err
	}

	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return snapshot, err
	}

	if snapshot.Ports == nil {
		snapshot.Ports = defaultFanSnapshot().Ports
	}

	for _, portKey := range []string{"port1", "port2", "port3", "port4"} {
		if _, ok := snapshot.Ports[portKey]; !ok {
			snapshot.Ports[portKey] = fanPortState{Mode: "unknown", UpdatedAt: time.Now().Format(time.RFC3339)}
		}
	}

	return snapshot, nil
}

func saveFanSnapshot(snapshot fanPortSnapshot) error {
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fanFilePath, payload, 0o644)
}

func saveFanSnapshotForPort(port int, source string, portState fanPortState) error {
	snapshot, err := loadFanSnapshot()
	if err != nil {
		return err
	}

	snapshot.Source = source
	snapshot.Ports[fmt.Sprintf("port%d", port)] = portState
	return saveFanSnapshot(snapshot)
}

func saveFanSnapshotForAllPorts(source string, portState fanPortState) error {
	snapshot, err := loadFanSnapshot()
	if err != nil {
		return err
	}

	snapshot.Source = source
	for _, portKey := range []string{"port1", "port2", "port3", "port4"} {
		snapshot.Ports[portKey] = portState
	}

	return saveFanSnapshot(snapshot)
}

func channelsForPort(channelMap hidPortChannelMap, port int) [2]int {
	switch port {
	case 1:
		return channelMap.Port1
	case 2:
		return channelMap.Port2
	case 3:
		return channelMap.Port3
	default:
		return channelMap.Port4
	}
}

func validateHIDPortChannelMap(channelMap hidPortChannelMap) error {
	all := [][2]int{channelMap.Port1, channelMap.Port2, channelMap.Port3, channelMap.Port4}
	for portIdx, pair := range all {
		if pair[0] < 0 || pair[0] > 7 || pair[1] < 0 || pair[1] > 7 {
			return fmt.Errorf("invalid channel map for port %d", portIdx+1)
		}
		if pair[0] == pair[1] {
			return fmt.Errorf("channel map for port %d must have two distinct channels", portIdx+1)
		}
	}
	return nil
}

func parseFanSpeedOrPreset(input string) (int, string, error) {
	clean := strings.ToLower(strings.TrimSpace(input))

	switch clean {
	case "quiet":
		return 35, "quiet", nil
	case "standard":
		return 55, "standard", nil
	case "performance":
		return 80, "performance", nil
	}

	speed, err := strconv.Atoi(clean)
	if err != nil || speed < 0 || speed > 100 {
		return 0, "", fmt.Errorf("fan-all value must be 0..100 or one of: quiet, standard, performance")
	}

	return speed, "", nil
}

func parseEffectCode(input string) (byte, string, error) {
	clean := strings.ToLower(strings.TrimSpace(input))
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, "_", "")

	effectByName := map[string]byte{
		"static":          0x01,
		"breathing":       0x02,
		"breath":          0x02,
		"rainbowmorph":    0x04,
		"spectrumcycle":   0x04,
		"rainbow":         0x05,
		"rainbowwave":     0x05,
		"staggered":       0x18,
		"tide":            0x1A,
		"runway":          0x1C,
		"mixing":          0x1E,
		"stack":           0x20,
		"stackmulticolor": 0x21,
		"neon":            0x22,
		"colorcycle":      0x23,
		"meteor":          0x24,
		"voice":           0x26,
		"groove":          0x27,
		"render":          0x28,
		"tunnel":          0x29,
		// Convenience aliases for additional UI effect names.
		"mopup":           0x2A,
		"scan":            0x2B,
		"door":            0x2C,
		"heartbeat":       0x2D,
		"heartbeatrunway": 0x2E,
		"disco":           0x2F,
		"electriccurrent": 0x30,
		"warning":         0x31,
	}

	if code, ok := effectByName[clean]; ok {
		return code, clean, nil
	}

	value, err := strconv.ParseInt(clean, 0, 64)
	if err != nil || value < 0 || value > 0xFF {
		return 0, "", fmt.Errorf("effect must be a known name or numeric 0..255/0x00..0xFF")
	}

	return byte(value), fmt.Sprintf("0x%02X", value), nil
}

// parseHexColor normalizes a #RRGGBB value and returns the uppercase hex string
// plus its RGB byte components.
func parseHexColor(input string) (string, uint8, uint8, uint8, error) {
	cleanHex := strings.TrimPrefix(strings.TrimSpace(input), "#")
	if len(cleanHex) == 6 {
		rgb, err := hex.DecodeString(cleanHex)
		if err == nil {
			return strings.ToUpper(cleanHex), rgb[0], rgb[1], rgb[2], nil
		}
	}

	hexName, red, green, blue, ok := parseNamedColor(input)
	if ok {
		return hexName, red, green, blue, nil
	}

	return "", 0, 0, 0, fmt.Errorf("color must be hex (#RRGGBB) or a named color like red, light red, dark blue")
}

func parseNamedColor(input string) (string, uint8, uint8, uint8, bool) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")

	baseColors := map[string][3]uint8{
		"red":     {255, 0, 0},
		"orange":  {255, 128, 0},
		"yellow":  {255, 220, 0},
		"green":   {0, 220, 80},
		"blue":    {0, 120, 255},
		"magenta": {255, 0, 180},
		"purple":  {140, 60, 220},
		"cyan":    {0, 220, 220},
		"pink":    {255, 105, 180},
		"white":   {255, 255, 255},
		"aqua":    {0, 220, 220},
		"teal":    {0, 170, 150},
		"lime":    {80, 255, 80},
		"violet":  {165, 90, 235},
		"indigo":  {75, 0, 180},
		"amber":   {255, 160, 0},
		"gray":    {160, 160, 160},
		"grey":    {160, 160, 160},
		"black":   {0, 0, 0},
	}

	if rgb, ok := baseColors[normalized]; ok {
		hexName := fmt.Sprintf("%02X%02X%02X", rgb[0], rgb[1], rgb[2])
		return hexName, rgb[0], rgb[1], rgb[2], true
	}

	applyShade := func(rgb [3]uint8, shade string) (uint8, uint8, uint8) {
		switch shade {
		case "light":
			return blendTowardWhite(rgb[0], 0.35), blendTowardWhite(rgb[1], 0.35), blendTowardWhite(rgb[2], 0.35)
		case "very light":
			return blendTowardWhite(rgb[0], 0.60), blendTowardWhite(rgb[1], 0.60), blendTowardWhite(rgb[2], 0.60)
		case "dark":
			return scaleDown(rgb[0], 0.65), scaleDown(rgb[1], 0.65), scaleDown(rgb[2], 0.65)
		case "very dark":
			return scaleDown(rgb[0], 0.40), scaleDown(rgb[1], 0.40), scaleDown(rgb[2], 0.40)
		default:
			return rgb[0], rgb[1], rgb[2]
		}
	}

	if strings.HasPrefix(normalized, "light ") {
		base := strings.TrimSpace(strings.TrimPrefix(normalized, "light "))
		if rgb, ok := baseColors[base]; ok {
			r, g, b := applyShade(rgb, "light")
			hexName := fmt.Sprintf("%02X%02X%02X", r, g, b)
			return hexName, r, g, b, true
		}
	}

	if strings.HasPrefix(normalized, "very light ") {
		base := strings.TrimSpace(strings.TrimPrefix(normalized, "very light "))
		if rgb, ok := baseColors[base]; ok {
			r, g, b := applyShade(rgb, "very light")
			hexName := fmt.Sprintf("%02X%02X%02X", r, g, b)
			return hexName, r, g, b, true
		}
	}

	if strings.HasPrefix(normalized, "dark ") {
		base := strings.TrimSpace(strings.TrimPrefix(normalized, "dark "))
		if rgb, ok := baseColors[base]; ok {
			r, g, b := applyShade(rgb, "dark")
			hexName := fmt.Sprintf("%02X%02X%02X", r, g, b)
			return hexName, r, g, b, true
		}
	}

	if strings.HasPrefix(normalized, "very dark ") {
		base := strings.TrimSpace(strings.TrimPrefix(normalized, "very dark "))
		if rgb, ok := baseColors[base]; ok {
			r, g, b := applyShade(rgb, "very dark")
			hexName := fmt.Sprintf("%02X%02X%02X", r, g, b)
			return hexName, r, g, b, true
		}
	}

	return "", 0, 0, 0, false
}

func blendTowardWhite(value uint8, amount float64) uint8 {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}
	blended := float64(value) + (255.0-float64(value))*amount
	if blended > 255 {
		blended = 255
	}
	return uint8(blended)
}

func scaleDown(value uint8, factor float64) uint8 {
	if factor < 0 {
		factor = 0
	}
	if factor > 1 {
		factor = 1
	}
	scaled := float64(value) * factor
	if scaled > 255 {
		scaled = 255
	}
	return uint8(scaled)
}
