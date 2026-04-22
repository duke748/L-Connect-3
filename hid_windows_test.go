//go:build windows

package main

import "testing"

func TestHIDBrightnessCode(t *testing.T) {
	tests := []struct {
		input int
		want  byte
	}{
		{input: -1, want: 0x08},
		{input: 0, want: 0x08},
		{input: 25, want: 0x03},
		{input: 50, want: 0x02},
		{input: 75, want: 0x01},
		{input: 100, want: 0x00},
	}

	for _, tc := range tests {
		if got := hidBrightnessCode(tc.input); got != tc.want {
			t.Fatalf("brightness %d -> got 0x%02X want 0x%02X", tc.input, got, tc.want)
		}
	}
}

func TestBuildStaticColorLEDData_OrderAndLength(t *testing.T) {
	data := buildStaticColorLEDData(10, 20, 30, 100)
	if len(data) != 80*3 {
		t.Fatalf("unexpected data length: got %d want %d", len(data), 80*3)
	}

	// Expected RBG order.
	if data[0] != 10 || data[1] != 30 || data[2] != 20 {
		t.Fatalf("unexpected first LED bytes: got [%d %d %d] want [10 30 20]", data[0], data[1], data[2])
	}
}

func TestBuildStaticColorLEDData_LimitsAndBrightness(t *testing.T) {
	full := buildStaticColorLEDData(255, 255, 255, 100)
	half := buildStaticColorLEDData(255, 255, 255, 50)

	// White is current-limited, then brightness-scaled. Ensure half is dimmer.
	if !(half[0] < full[0] && half[1] < full[1] && half[2] < full[2]) {
		t.Fatalf("expected half brightness to be dimmer: full=[%d %d %d] half=[%d %d %d]", full[0], full[1], full[2], half[0], half[1], half[2])
	}
}

func TestBuildPaletteLEDData_DistributesByFan(t *testing.T) {
	colors := []effectColor{
		{R: 255, G: 0, B: 0},
		{R: 0, G: 255, B: 0},
	}
	data := buildPaletteLEDData(colors, 100)

	if len(data) != 80*3 {
		t.Fatalf("unexpected data length: got %d", len(data))
	}

	// LED 0 (fan 0) should use first color in RBG order: red => [255,0,0].
	if data[0] != 255 || data[1] != 0 || data[2] != 0 {
		t.Fatalf("unexpected LED0 bytes: [%d %d %d]", data[0], data[1], data[2])
	}

	// LED 16 is first LED of fan 1, should use second color (green) in RBG order: [0,0,255].
	off := 16 * 3
	if data[off] != 0 || data[off+1] != 0 || data[off+2] != 255 {
		t.Fatalf("unexpected LED16 bytes: [%d %d %d]", data[off], data[off+1], data[off+2])
	}
}

func TestScaleEffectColor(t *testing.T) {
	r, b, g := scaleEffectColor(effectColor{R: 100, G: 50, B: 25}, 100)
	if r != 100 || b != 25 || g != 50 {
		t.Fatalf("unexpected full brightness: got (%d,%d,%d) want (100,25,50)", r, b, g)
	}

	r, b, g = scaleEffectColor(effectColor{R: 100, G: 50, B: 25}, 50)
	if !(r < 100 && b < 25 && g < 50) {
		t.Fatalf("expected dimmed values at 50%%, got (%d,%d,%d)", r, b, g)
	}
}
