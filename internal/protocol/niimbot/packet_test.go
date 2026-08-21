package niimbot

import (
	"encoding/hex"
	"testing"
)

func TestConnectPacket(t *testing.T) {
	want := "035555c10101c1aaaa"
	if got := hex.EncodeToString(ConnectPacket()); got != want {
		t.Fatalf("ConnectPacket() = %s, want %s", got, want)
	}
}

func TestBasicControlPackets(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want string
	}{
		{name: "heartbeat", got: HeartbeatPacket(), want: "5555dc0101dcaaaa"},
		{name: "print_status", got: PrintStatusPacket(), want: "5555a30101a3aaaa"},
		{name: "printer_info", got: PrinterInfoPacket(), want: "5555a50101a5aaaa"},
		{name: "set_density", got: SetDensityPacket(0x02), want: "555521010222aaaa"},
		{name: "set_label_type", got: SetLabelTypePacket(0x01), want: "555523010123aaaa"},
		{name: "print_start", got: PrintStartPacket(), want: "555501010101aaaa"},
		{name: "print_clear", got: PrintClearPacket(), want: "555520010120aaaa"},
		{name: "page_start", got: PageStartPacket(), want: "555503010103aaaa"},
		{name: "print_quantity", got: PrintQuantityPacket(1), want: "55551502000116aaaa"},
		{name: "page_end", got: PageEndPacket(), want: "5555e30101e3aaaa"},
		{name: "print_end", got: PrintEndPacket(), want: "5555f30101f3aaaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hex.EncodeToString(tt.got); got != tt.want {
				t.Fatalf("%s = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestPageSizePackets(t *testing.T) {
	if got, want := hex.EncodeToString(SetPageSizePacket(128, 96)), "5555130400800060f7aaaa"; got != want {
		t.Fatalf("SetPageSizePacket() = %s, want %s", got, want)
	}

	if got, want := hex.EncodeToString(PrintStartV4Packet(1, 0, 1)), "5555010900010000000000010008aaaa"; got != want {
		t.Fatalf("PrintStartV4Packet() = %s, want %s", got, want)
	}

	if got, want := hex.EncodeToString(PrintStartPagesPacket(1, 0)), "555501070001000000000007aaaa"; got != want {
		t.Fatalf("PrintStartPagesPacket() = %s, want %s", got, want)
	}

	if got, want := hex.EncodeToString(SetPageSizeV4Packet(320, 96, 1)), "5555130d014000600001000000000000003eaaaa"; got != want {
		t.Fatalf("SetPageSizeV4Packet() = %s, want %s", got, want)
	}

	if got, want := hex.EncodeToString(SetPageSizePagesPacket(320, 320, 1)), "5555130601400140000114aaaa"; got != want {
		t.Fatalf("SetPageSizePagesPacket() = %s, want %s", got, want)
	}
}

func TestRowPackets(t *testing.T) {
	row := mustDecodeHex(t, "00fffffe070038fc7fffff00")
	if got, want := hex.EncodeToString(BitmapRowPacket(8, row)), "555585120008170c170100fffffe070038fc7fffff00d0aaaa"; got != want {
		t.Fatalf("BitmapRowPacket() = %s, want %s", got, want)
	}

	indexedRow := mustDecodeHex(t, "000000000003f00000000000")
	if got, want := hex.EncodeToString(BitmapRowIndexedPacket(0x73, indexedRow)), "55558312007300060001002e002f0030003100320033e4aaaa"; got != want {
		t.Fatalf("BitmapRowIndexedPacket() = %s, want %s", got, want)
	}

	if got, want := hex.EncodeToString(EmptyRowPacket(0, 1)), "5555840300000186aaaa"; got != want {
		t.Fatalf("EmptyRowPacket() = %s, want %s", got, want)
	}
}

func TestParseFramedPacket(t *testing.T) {
	packet, err := ParseFramedPacket(mustDecodeHex(t, "5555b3080001646423260001beaaaa"))
	if err != nil {
		t.Fatalf("ParseFramedPacket() error = %v", err)
	}
	if packet.Command != 0xb3 {
		t.Fatalf("command = 0x%02x, want 0xb3", packet.Command)
	}
	if got, want := hex.EncodeToString(packet.Payload), "0001646423260001"; got != want {
		t.Fatalf("payload = %s, want %s", got, want)
	}
}

func mustDecodeHex(t *testing.T, value string) []byte {
	t.Helper()
	b, err := hex.DecodeString(value)
	if err != nil {
		t.Fatalf("decode hex %q: %v", value, err)
	}
	return b
}
