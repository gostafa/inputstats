//go:build linux

package linux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/holoplot/go-evdev"
	"golang.org/x/sys/unix"
)

func classifyDevice(dev evdevInfo) deviceClass {
	if hasAccelProp(dev) {
		return deviceClass{}
	}

	var class deviceClass

	keyCodes := codeSet(dev.CapableEvents(evdev.EV_KEY))
	relCodes := codeSet(dev.CapableEvents(evdev.EV_REL))

	if hasKeyboardKeys(keyCodes) {
		class.keyboard = true
	}

	if hasMouseAxes(relCodes) || keyCodes[evdev.BTN_LEFT] {
		class.mouse = true
	}

	return class
}

func codeSet(codes []evdev.EvCode) map[evdev.EvCode]bool {
	set := make(map[evdev.EvCode]bool, len(codes))
	for index := range codes {
		set[codes[index]] = true
	}

	return set
}

func hasAccelProp(dev evdevInfo) bool {
	if hasAccelProperty(dev) {
		return true
	}

	return isSkipDeviceName(dev)
}

func hasAccelProperty(dev evdevInfo) bool {
	return slices.Contains(dev.Properties(), evdev.INPUT_PROP_ACCELEROMETER)
}

func hasKeyboardKeys(keys map[evdev.EvCode]bool) bool {
	if keys[evdev.KEY_SPACE] {
		return true
	}

	for code := evdev.EvCode(evdev.KEY_A); code <= evdev.KEY_Z; code++ {
		if keys[code] {
			return true
		}
	}

	return false
}

func hasMouseAxes(rel map[evdev.EvCode]bool) bool {
	return rel[evdev.REL_X] || rel[evdev.REL_Y]
}

func isAgain(err error) bool {
	return errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) ||
		errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK)
}

func isSkipDeviceName(dev evdevInfo) bool {
	name, err := dev.Name()
	if err != nil {
		return false
	}

	return nameLooksSkipped(strings.ToLower(name))
}

func listEventPaths() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(inputDir, eventNameGlob))
	if err != nil {
		return nil, fmt.Errorf(errFmtWrap, err)
	}

	return matches, nil
}

func mapListErr(err error) error {
	if errors.Is(err, os.ErrPermission) || os.IsPermission(err) {
		return domain.ErrPermissionDenied
	}

	if errors.Is(err, os.ErrNotExist) {
		return domain.ErrNoInputDevices
	}

	return fmt.Errorf(errFmtWrap, err)
}

func nameLooksSkipped(lower string) bool {
	if strings.Contains(lower, "accelerometer") {
		return true
	}

	if strings.Contains(lower, "lid") && !strings.Contains(lower, "keyboard") {
		return true
	}

	return strings.Contains(lower, "power button") ||
		strings.Contains(lower, "sleep button")
}

func noteProbeErr(err error, stats *probeStats) {
	if errors.Is(err, errDeviceSkipped) {
		return
	}

	if errors.Is(err, os.ErrPermission) || os.IsPermission(err) {
		stats.sawPermission = true
	}
}

func probeAll(paths []string) probeStats {
	var stats probeStats

	for index := range paths {
		probeOne(paths[index], &stats)
	}

	return stats
}

func probeDevices() error {
	paths, err := listEventPaths()
	if err != nil {
		return fmt.Errorf(errFmtWrap, mapListErr(err))
	}

	if len(paths) == evSyn {
		return domain.ErrNoInputDevices
	}

	err = probeOutcome(probeAll(paths))
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func probeOne(path string, stats *probeStats) {
	dev, class, err := openAndClassify(path)
	if err != nil {
		noteProbeErr(err, stats)

		return
	}

	if !usable(class) {
		closeHandle(dev)

		return
	}

	stats.usable++

	closeHandle(dev)
}

func probeOutcome(stats probeStats) error {
	if stats.usable > evSyn {
		return nil
	}

	if stats.sawPermission {
		return domain.ErrPermissionDenied
	}

	return domain.ErrNoInputDevices
}
