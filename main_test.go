package main

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func withTempWorkingDir(t *testing.T) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir to temp dir failed: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory failed: %v", err)
		}
	})
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe failed: %v", err)
	}

	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close write pipe failed: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout pipe failed: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close read pipe failed: %v", err)
	}

	return string(out)
}

func TestParseFanSpeedOrPreset(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSpeed  int
		wantPreset string
		wantErr    bool
	}{
		{name: "quiet preset", input: "quiet", wantSpeed: 35, wantPreset: "quiet"},
		{name: "standard preset", input: "standard", wantSpeed: 55, wantPreset: "standard"},
		{name: "performance preset", input: "performance", wantSpeed: 80, wantPreset: "performance"},
		{name: "manual speed", input: "42", wantSpeed: 42, wantPreset: ""},
		{name: "invalid high", input: "101", wantErr: true},
		{name: "invalid text", input: "turbo", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotSpeed, gotPreset, err := parseFanSpeedOrPreset(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotSpeed != tc.wantSpeed {
				t.Fatalf("speed mismatch: got %d want %d", gotSpeed, tc.wantSpeed)
			}
			if gotPreset != tc.wantPreset {
				t.Fatalf("preset mismatch: got %q want %q", gotPreset, tc.wantPreset)
			}
		})
	}
}

func TestParseEffectCode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCode  byte
		wantLabel string
		wantErr   bool
	}{
		{name: "named effect", input: "breathing", wantCode: 0x02, wantLabel: "breathing"},
		{name: "normalized alias", input: "rainbow-wave", wantCode: 0x05, wantLabel: "rainbowwave"},
		{name: "hex value", input: "0x24", wantCode: 0x24, wantLabel: "0x24"},
		{name: "decimal value", input: "36", wantCode: 0x24, wantLabel: "0x24"},
		{name: "invalid", input: "not-an-effect", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotCode, gotLabel, err := parseEffectCode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotCode != tc.wantCode {
				t.Fatalf("code mismatch: got 0x%02X want 0x%02X", gotCode, tc.wantCode)
			}
			if gotLabel != tc.wantLabel {
				t.Fatalf("label mismatch: got %q want %q", gotLabel, tc.wantLabel)
			}
		})
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHex   string
		wantR     uint8
		wantG     uint8
		wantB     uint8
		wantError bool
	}{
		{name: "hex with hash", input: "#ff6600", wantHex: "FF6600", wantR: 255, wantG: 102, wantB: 0},
		{name: "named color", input: "teal", wantHex: "00AA96", wantR: 0, wantG: 170, wantB: 150},
		{name: "light red", input: "light red", wantHex: "FF5959", wantR: 255, wantG: 89, wantB: 89},
		{name: "very dark orange", input: "very dark orange", wantHex: "663300", wantR: 102, wantG: 51, wantB: 0},
		{name: "invalid", input: "bluish", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotHex, r, g, b, err := parseHexColor(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotHex != tc.wantHex {
				t.Fatalf("hex mismatch: got %q want %q", gotHex, tc.wantHex)
			}
			if r != tc.wantR || g != tc.wantG || b != tc.wantB {
				t.Fatalf("rgb mismatch: got (%d,%d,%d) want (%d,%d,%d)", r, g, b, tc.wantR, tc.wantG, tc.wantB)
			}
		})
	}
}

func TestParseHexColorList(t *testing.T) {
	colors, hexes, err := parseHexColorList("#FF0000, dark blue", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(colors) != 2 || len(hexes) != 2 {
		t.Fatalf("expected 2 colors, got colors=%d hexes=%d", len(colors), len(hexes))
	}
	if hexes[0] != "FF0000" {
		t.Fatalf("first color mismatch: got %q", hexes[0])
	}
	if hexes[1] != "004EA5" {
		t.Fatalf("second color mismatch: got %q", hexes[1])
	}

	_, _, err = parseHexColorList("", 4)
	if err == nil {
		t.Fatalf("expected error for empty list")
	}

	_, _, err = parseHexColorList("red,green,blue,white,orange", 4)
	if err == nil {
		t.Fatalf("expected error for max color overflow")
	}
}

func TestTargetPortsForEffect(t *testing.T) {
	all := targetPortsForEffect(0)
	if len(all) != 4 || all[0] != 1 || all[3] != 4 {
		t.Fatalf("unexpected all ports result: %v", all)
	}

	one := targetPortsForEffect(3)
	if len(one) != 1 || one[0] != 3 {
		t.Fatalf("unexpected single port result: %v", one)
	}
}

func TestRawChannelPortLabel(t *testing.T) {
	tests := map[int]string{
		0: "port1",
		1: "port1",
		2: "port2",
		5: "port3",
		7: "port4",
		8: "",
	}

	for channel, want := range tests {
		if got := rawChannelPortLabel(channel); got != want {
			t.Fatalf("channel %d -> got %q want %q", channel, got, want)
		}
	}
}

func TestValidateHIDPortChannelMap(t *testing.T) {
	valid := hidPortChannelMap{
		Port1: [2]int{0, 1},
		Port2: [2]int{2, 3},
		Port3: [2]int{4, 5},
		Port4: [2]int{6, 7},
	}
	if err := validateHIDPortChannelMap(valid); err != nil {
		t.Fatalf("unexpected error for valid map: %v", err)
	}

	dup := valid
	dup.Port2 = [2]int{2, 2}
	if err := validateHIDPortChannelMap(dup); err == nil {
		t.Fatalf("expected error for duplicate channel pair")
	}

	oob := valid
	oob.Port4 = [2]int{6, 8}
	if err := validateHIDPortChannelMap(oob); err == nil {
		t.Fatalf("expected error for out-of-range channel")
	}
}

func TestLoadHIDPortChannelMap_DefaultWhenMissing(t *testing.T) {
	withTempWorkingDir(t)

	got, err := loadHIDPortChannelMap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := defaultHIDPortChannelMap()
	if got != want {
		t.Fatalf("default channel map mismatch: got %+v want %+v", got, want)
	}
}

func TestSaveAndLoadHIDPortChannelMap_RoundTrip(t *testing.T) {
	withTempWorkingDir(t)

	want := hidPortChannelMap{
		Port1: [2]int{1, 0},
		Port2: [2]int{3, 2},
		Port3: [2]int{5, 4},
		Port4: [2]int{7, 6},
	}

	if err := saveHIDPortChannelMap(want); err != nil {
		t.Fatalf("save map failed: %v", err)
	}

	got, err := loadHIDPortChannelMap()
	if err != nil {
		t.Fatalf("load map failed: %v", err)
	}

	if got != want {
		t.Fatalf("round-trip map mismatch: got %+v want %+v", got, want)
	}
}

func TestSaveAndLoadFanTargetState_RoundTrip(t *testing.T) {
	withTempWorkingDir(t)

	want := fanTargetState{
		Source:    "test",
		Mode:      "manual",
		Speed:     62,
		Preset:    "",
		UpdatedAt: "2026-04-22T12:34:56Z",
	}

	if err := saveFanTargetState(want); err != nil {
		t.Fatalf("save fan target failed: %v", err)
	}

	got, err := loadFanTargetState()
	if err != nil {
		t.Fatalf("load fan target failed: %v", err)
	}

	if got != want {
		t.Fatalf("round-trip fan target mismatch: got %+v want %+v", got, want)
	}
}

func TestSaveLightingStateForPortAndAll(t *testing.T) {
	withTempWorkingDir(t)

	portState := lightingPortState{
		Mode:       "static",
		EffectCode: "0x01",
		Layout:     "single",
		Color:      "FF6600",
		Colors:     []string{"FF6600"},
		Brightness: 75,
		UpdatedAt:  "2026-04-22T12:00:00Z",
	}

	if err := saveLightingStateForPort(2, "test-port", portState); err != nil {
		t.Fatalf("save lighting state for port failed: %v", err)
	}

	state, err := loadLightingState()
	if err != nil {
		t.Fatalf("load lighting state failed: %v", err)
	}
	if state.Source != "test-port" {
		t.Fatalf("unexpected source after port save: %q", state.Source)
	}
	if state.Ports["port2"].Mode != "static" {
		t.Fatalf("expected port2 mode static, got %q", state.Ports["port2"].Mode)
	}

	allState := lightingPortState{
		Mode:       "breathing",
		EffectCode: "0x02",
		Layout:     "single",
		Brightness: 50,
		UpdatedAt:  "2026-04-22T13:00:00Z",
	}
	if err := saveLightingStateForAllPorts("test-all", allState); err != nil {
		t.Fatalf("save lighting state for all ports failed: %v", err)
	}

	state, err = loadLightingState()
	if err != nil {
		t.Fatalf("load lighting state failed: %v", err)
	}
	if state.Source != "test-all" {
		t.Fatalf("unexpected source after all-port save: %q", state.Source)
	}
	for _, key := range []string{"port1", "port2", "port3", "port4"} {
		if state.Ports[key].Mode != "breathing" {
			t.Fatalf("expected %s mode breathing, got %q", key, state.Ports[key].Mode)
		}
	}
}

func TestSaveFanSnapshotForPortAndAll(t *testing.T) {
	withTempWorkingDir(t)

	portState := fanPortState{
		Mode:      "manual",
		Speed:     45,
		UpdatedAt: "2026-04-22T12:00:00Z",
	}
	if err := saveFanSnapshotForPort(3, "test-port", portState); err != nil {
		t.Fatalf("save fan snapshot for port failed: %v", err)
	}

	snapshot, err := loadFanSnapshot()
	if err != nil {
		t.Fatalf("load fan snapshot failed: %v", err)
	}
	if snapshot.Source != "test-port" {
		t.Fatalf("unexpected source after port save: %q", snapshot.Source)
	}
	if snapshot.Ports["port3"].Speed != 45 {
		t.Fatalf("expected port3 speed 45, got %d", snapshot.Ports["port3"].Speed)
	}

	allState := fanPortState{
		Mode:      "preset",
		Speed:     80,
		Preset:    "performance",
		UpdatedAt: "2026-04-22T13:00:00Z",
	}
	if err := saveFanSnapshotForAllPorts("test-all", allState); err != nil {
		t.Fatalf("save fan snapshot for all ports failed: %v", err)
	}

	snapshot, err = loadFanSnapshot()
	if err != nil {
		t.Fatalf("load fan snapshot failed: %v", err)
	}
	if snapshot.Source != "test-all" {
		t.Fatalf("unexpected source after all-port save: %q", snapshot.Source)
	}
	for _, key := range []string{"port1", "port2", "port3", "port4"} {
		if snapshot.Ports[key].Speed != 80 || snapshot.Ports[key].Mode != "preset" {
			t.Fatalf("expected %s preset speed 80, got mode=%q speed=%d", key, snapshot.Ports[key].Mode, snapshot.Ports[key].Speed)
		}
	}
}

func TestRunHIDFan_UsesParsedValues(t *testing.T) {
	withTempWorkingDir(t)

	original := hidFanSetFunc
	t.Cleanup(func() { hidFanSetFunc = original })

	called := 0
	var gotVID uint16
	var gotPID uint16
	var gotPort int
	var gotSpeed int
	hidFanSetFunc = func(vendorID uint16, productID uint16, port int, speed int) error {
		called++
		gotVID = vendorID
		gotPID = productID
		gotPort = port
		gotSpeed = speed
		return nil
	}

	if err := runHIDFan("2", "60"); err != nil {
		t.Fatalf("runHIDFan failed: %v", err)
	}

	if called != 1 {
		t.Fatalf("expected one HID call, got %d", called)
	}
	if gotVID != slInfinityVID || gotPID != slInfinityPID || gotPort != 2 || gotSpeed != 60 {
		t.Fatalf("unexpected HID call args: vid=0x%04X pid=0x%04X port=%d speed=%d", gotVID, gotPID, gotPort, gotSpeed)
	}
}

func TestRunHIDFan_PropagatesHIDError(t *testing.T) {
	withTempWorkingDir(t)

	original := hidFanSetFunc
	t.Cleanup(func() { hidFanSetFunc = original })

	wantErr := errors.New("hid write failed")
	hidFanSetFunc = func(vendorID uint16, productID uint16, port int, speed int) error {
		return wantErr
	}

	err := runHIDFan("1", "50")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped HID error, got %v", err)
	}
}

func TestRunHIDSetChannel_UsesParsedColor(t *testing.T) {
	withTempWorkingDir(t)

	original := hidSetStaticColorChannelByIDFunc
	t.Cleanup(func() { hidSetStaticColorChannelByIDFunc = original })

	called := 0
	var gotChannel int
	var gotR uint8
	var gotG uint8
	var gotB uint8
	var gotBrightness int
	hidSetStaticColorChannelByIDFunc = func(vendorID uint16, productID uint16, channel int, red uint8, green uint8, blue uint8, brightnessPct int) error {
		called++
		gotChannel = channel
		gotR = red
		gotG = green
		gotB = blue
		gotBrightness = brightnessPct
		return nil
	}

	if err := runHIDSetChannel("3", "#FF6600", "75"); err != nil {
		t.Fatalf("runHIDSetChannel failed: %v", err)
	}

	if called != 1 {
		t.Fatalf("expected one HID call, got %d", called)
	}
	if gotChannel != 3 || gotR != 255 || gotG != 102 || gotB != 0 || gotBrightness != 75 {
		t.Fatalf("unexpected HID args: channel=%d rgb=(%d,%d,%d) brightness=%d", gotChannel, gotR, gotG, gotB, gotBrightness)
	}
}

func TestRunHIDEffect_Port0CallsEvenChannels(t *testing.T) {
	withTempWorkingDir(t)

	original := hidApplyEffectChannelByIDFunc
	t.Cleanup(func() { hidApplyEffectChannelByIDFunc = original })

	channels := make([]int, 0, 4)
	hidApplyEffectChannelByIDFunc = func(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, hasColor bool, red uint8, green uint8, blue uint8) error {
		channels = append(channels, channel)
		if hasColor {
			t.Fatalf("expected hasColor=false for empty color arg")
		}
		return nil
	}

	if err := runHIDEffect("breathing", "", 0, 2, 80, 1); err != nil {
		t.Fatalf("runHIDEffect failed: %v", err)
	}

	want := []int{0, 2, 4, 6}
	if !reflect.DeepEqual(channels, want) {
		t.Fatalf("unexpected channels: got %v want %v", channels, want)
	}
}

func TestRunHIDEffectLinked_PortSpecificCallsMappedChannels(t *testing.T) {
	withTempWorkingDir(t)

	channelMap := hidPortChannelMap{
		Port1: [2]int{0, 1},
		Port2: [2]int{4, 5},
		Port3: [2]int{2, 3},
		Port4: [2]int{6, 7},
	}
	if err := saveHIDPortChannelMap(channelMap); err != nil {
		t.Fatalf("save map failed: %v", err)
	}

	original := hidApplyEffectPaletteChannelByIDFn
	t.Cleanup(func() { hidApplyEffectPaletteChannelByIDFn = original })

	channels := make([]int, 0, 2)
	hidApplyEffectPaletteChannelByIDFn = func(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, colors []effectColor) error {
		channels = append(channels, channel)
		return nil
	}

	err := runHIDEffectLinked("mixing", "#FF0000,#0000FF", 2, 1, 90, 0)
	if err != nil {
		t.Fatalf("runHIDEffectLinked failed: %v", err)
	}

	want := []int{4, 5}
	if !reflect.DeepEqual(channels, want) {
		t.Fatalf("unexpected linked channels: got %v want %v", channels, want)
	}
}

func TestRunHIDEffectSplit_PortSpecificCallsMappedPrimaryThenSecondary(t *testing.T) {
	withTempWorkingDir(t)

	channelMap := hidPortChannelMap{
		Port1: [2]int{0, 1},
		Port2: [2]int{6, 7},
		Port3: [2]int{2, 3},
		Port4: [2]int{4, 5},
	}
	if err := saveHIDPortChannelMap(channelMap); err != nil {
		t.Fatalf("save map failed: %v", err)
	}

	original := hidApplyEffectPaletteChannelByIDFn
	t.Cleanup(func() { hidApplyEffectPaletteChannelByIDFn = original })

	channels := make([]int, 0, 2)
	hidApplyEffectPaletteChannelByIDFn = func(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, colors []effectColor) error {
		channels = append(channels, channel)
		return nil
	}

	err := runHIDEffectSplit("door", "meteor", "#FF8800,#FFFFFF", "#00D5FF,#FF4D7A", 2, 2, 100, 0)
	if err != nil {
		t.Fatalf("runHIDEffectSplit failed: %v", err)
	}

	want := []int{6, 7}
	if !reflect.DeepEqual(channels, want) {
		t.Fatalf("unexpected split channels: got %v want %v", channels, want)
	}
}

func TestRunHIDEffectSplit_PropagatesPrimaryError(t *testing.T) {
	withTempWorkingDir(t)

	original := hidApplyEffectPaletteChannelByIDFn
	t.Cleanup(func() { hidApplyEffectPaletteChannelByIDFn = original })

	wantErr := errors.New("primary write failed")
	hidApplyEffectPaletteChannelByIDFn = func(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, colors []effectColor) error {
		return wantErr
	}

	err := runHIDEffectSplit("door", "meteor", "#FF8800", "#00D5FF", 1, 2, 100, 0)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped primary error, got %v", err)
	}
}

func TestRunHIDEffectSplit_PropagatesSecondaryError(t *testing.T) {
	withTempWorkingDir(t)

	original := hidApplyEffectPaletteChannelByIDFn
	t.Cleanup(func() { hidApplyEffectPaletteChannelByIDFn = original })

	wantErr := errors.New("secondary write failed")
	callCount := 0
	hidApplyEffectPaletteChannelByIDFn = func(vendorID uint16, productID uint16, channel int, effect byte, speed int, direction int, brightnessPct int, colors []effectColor) error {
		callCount++
		if callCount == 2 {
			return wantErr
		}
		return nil
	}

	err := runHIDEffectSplit("door", "meteor", "#FF8800", "#00D5FF", 1, 2, 100, 0)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped secondary error, got %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected two calls before failure, got %d", callCount)
	}
}

func TestRunHIDList_PrintsTableOutput(t *testing.T) {
	original := hidEnumerateDevicesFunc
	t.Cleanup(func() { hidEnumerateDevicesFunc = original })

	hidEnumerateDevicesFunc = func() ([]hidDeviceEntry, error) {
		return []hidDeviceEntry{
			{
				VendorID:           0x0CF2,
				ProductID:          0xA102,
				Usage:              0x00A1,
				UsagePage:          0xFF72,
				InterfaceNumber:    1,
				ManufacturerString: "ENE",
				ProductString:      "LianLi-SL-infinity-v1.4",
			},
		}, nil
	}

	output := captureStdout(t, func() {
		if err := runHIDList(); err != nil {
			t.Fatalf("runHIDList failed: %v", err)
		}
	})

	for _, expected := range []string{
		"VID",
		"PID",
		"Manufacturer",
		"Product",
		"0x0CF2",
		"0xA102",
		"ENE",
		"LianLi-SL-infinity-v1.4",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestRunHIDList_PrintsNoDevicesMessage(t *testing.T) {
	original := hidEnumerateDevicesFunc
	t.Cleanup(func() { hidEnumerateDevicesFunc = original })

	hidEnumerateDevicesFunc = func() ([]hidDeviceEntry, error) {
		return nil, nil
	}

	output := captureStdout(t, func() {
		if err := runHIDList(); err != nil {
			t.Fatalf("runHIDList failed: %v", err)
		}
	})

	if !strings.Contains(output, "no HID devices found") {
		t.Fatalf("expected no-devices message, got:\n%s", output)
	}
}

func TestRunHIDStatus_PrintsRPMAndLastTarget(t *testing.T) {
	withTempWorkingDir(t)

	original := hidReadRPMFunc
	t.Cleanup(func() { hidReadRPMFunc = original })

	hidReadRPMFunc = func(vendorID uint16, productID uint16) ([4]uint16, error) {
		return [4]uint16{1234, 2345, 3456, 4567}, nil
	}

	if err := saveFanTargetState(fanTargetState{
		Source:    "fan-all",
		Mode:      "preset",
		Speed:     80,
		Preset:    "performance",
		UpdatedAt: "2026-04-22T13:00:00Z",
	}); err != nil {
		t.Fatalf("save fan target state failed: %v", err)
	}

	output := captureStdout(t, func() {
		if err := runHIDStatus(); err != nil {
			t.Fatalf("runHIDStatus failed: %v", err)
		}
	})

	for _, expected := range []string{
		"hid status",
		"port 1: 1234 RPM",
		"port 4: 4567 RPM",
		"last target: fan-all preset=\"performance\" speed=80% at 2026-04-22T13:00:00Z",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}
