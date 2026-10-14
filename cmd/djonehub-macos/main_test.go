package main

import (
	"fmt"
	"testing"
)

func TestParseUSBModem(t *testing.T) {
	for _, tt := range []struct {
		name                string
		vendorID, productID int
		want                bool
	}{
		{"DJI", 0x2ca3, 0x4006, true},
		{"BAIWANG", 0x2c7c, 0x0125, true},
		{"other DJI product", 0x2ca3, 0x0001, false},
		{"other Quectel product", 0x2c7c, 0x0001, false},
		{"unrelated vendor", 0x05ac, 0x0125, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			inventory := fmt.Sprintf(`
"idVendor" = %d
"idProduct" = %d
"USB Vendor Name" = "%s"
"USB Product Name" = "EG25G-QDC507"
"locationID" = 17825792
"bInterfaceNumber" = 2
"bInterfaceClass" = 255
"bNumEndpoints" = 3
`, tt.vendorID, tt.productID, tt.name)
			device := parseUSBModem(inventory)
			if !tt.want {
				if device != nil {
					t.Fatalf("unsupported device discovered: %+v", device)
				}
				return
			}
			if device == nil || device.Vendor != tt.name || device.VendorID != fmt.Sprintf("%04x", tt.vendorID) || device.ProductID != fmt.Sprintf("%04x", tt.productID) {
				t.Fatalf("incorrect USB identity: %+v", device)
			}
			if len(device.Interfaces) != 1 || device.Interfaces[0].Number != 2 || device.Interfaces[0].Endpoints != 3 {
				t.Fatalf("incorrect USB interfaces: %+v", device.Interfaces)
			}
		})
	}
}

func TestPortScore(t *testing.T) {
	tests := []struct {
		name string
		port string
		want int
	}{
		{name: "named Quectel port", port: "/dev/cu.Quectel-AT", want: 100},
		{name: "usb modem", port: "/dev/cu.usbmodem2101", want: 80},
		{name: "usb serial", port: "/dev/cu.usbserial-1420", want: 60},
		{name: "bluetooth", port: "/dev/cu.Bluetooth-Incoming-Port", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := portScore(tt.port); got != tt.want {
				t.Fatalf("portScore(%q) = %d, want %d", tt.port, got, tt.want)
			}
		})
	}
}

func TestParseUSBNetMode(t *testing.T) {
	for _, tt := range []struct {
		response string
		want     string
	}{
		{response: "AT+QCFG=\"usbnet\"\r\n+QCFG: \"usbnet\",0\r\nOK", want: "0"},
		{response: "+QCFG: \"usbnet\",1\r\nOK", want: "1"},
		{response: "ERROR", want: ""},
	} {
		if got := parseUSBNetMode(tt.response); got != tt.want {
			t.Fatalf("parseUSBNetMode(%q) = %q, want %q", tt.response, got, tt.want)
		}
	}
}

func TestParseUSBATOperator(t *testing.T) {
	for _, tt := range []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "known numeric PLMN",
			response: "AT+COPS?\r\n+COPS: 0,2,\"46015\",7\r\nOK",
			want:     "中国广电",
		},
		{
			name:     "long operator name",
			response: "+COPS: 0,0,\"CHN-UNICOM\",7\r\nOK",
			want:     "CHN-UNICOM",
		},
		{
			name:     "missing operator",
			response: "+COPS: 0\r\nOK",
			want:     "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseUSBATOperator(tt.response); got != tt.want {
				t.Fatalf("parseUSBATOperator(%q) = %q, want %q", tt.response, got, tt.want)
			}
		})
	}
}

func TestInitUSBATESIMManagerAfterDelayedUSBOpen(t *testing.T) {
	instance := &app{}

	manager, switchAllowed := instance.currentESIMManager()
	if manager != nil || switchAllowed {
		t.Fatalf("initial eSIM state = (%v, %v), want unavailable", manager, switchAllowed)
	}

	instance.initUSBATESIMManager()
	manager, switchAllowed = instance.currentESIMManager()
	if manager == nil {
		t.Fatal("USB AT recovery did not initialize the eSIM manager")
	}
	if !switchAllowed {
		t.Fatal("USB AT eSIM manager should allow profile switching")
	}

	instance.initUSBATESIMManager()
	managerAgain, _ := instance.currentESIMManager()
	if managerAgain != manager {
		t.Fatal("repeated USB AT recovery replaced the existing eSIM manager")
	}
}
