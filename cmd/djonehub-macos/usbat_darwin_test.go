//go:build darwin && cgo

package main

import (
	"os"
	"testing"
	"time"
)

func TestUSBATHardware(t *testing.T) {
	if os.Getenv("DJONEHUB_TEST_USB_AT") != "1" {
		t.Skip("set DJONEHUB_TEST_USB_AT=1 with a supported modem connected")
	}
	device := discoverUSBModem()
	if device == nil {
		t.Fatal("supported modem not discovered")
	}
	t.Logf("discovered %s %s (%s:%s)", device.Vendor, device.Product, device.VendorID, device.ProductID)
	instance := &app{}
	defer func() {
		if instance.usbAT != nil {
			instance.usbAT.Close()
		}
	}()
	for attempt := 0; attempt < 2; attempt++ {
		if err := instance.ensureUSBAT(); err != nil {
			t.Fatal(err)
		}
		t.Log(instance.usbAT.Description())
		for _, command := range []string{"AT", "ATI", `AT+QCFG="usbnet"`} {
			response, err := instance.usbAT.Command(command, 3*time.Second)
			if err != nil || !atProbeSucceeded(response) {
				t.Fatalf("%s: response=%q err=%v", command, response, err)
			}
			t.Logf("%s: %s", command, response)
		}
		instance.markUSBATDetached("hardware test closing handle to check reopening")
		if attempt == 0 {
			time.Sleep(time.Until(instance.usbATBackoffUntil))
		}
	}
}
