//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"

	"github.com/holoplot/go-evdev"
)

func alreadyOpen(reg *registry, path string) bool {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	return reg.opened[path] != nil
}

func armDevice(ctx context.Context, path string, claim *deviceClaim) bool {
	err := claim.dev.NonBlock()
	if err != nil {
		closeHandle(claim.dev)

		return false
	}

	return registerDevice(ctx, path, claim)
}

func openAndClassify(path string) (*evdev.InputDevice, deviceClass, error) {
	dev, err := evdev.OpenWithFlags(path, os.O_RDONLY)
	if err != nil {
		return nil, deviceClass{}, fmt.Errorf(errFmtWrap, err)
	}

	class := classifyDevice(dev)
	if !usable(class) {
		closeHandle(dev)

		return nil, class, errDeviceSkipped
	}

	return dev, class, nil
}

func openUsable(path string) (*evdev.InputDevice, deviceClass) {
	dev, class, err := openAndClassify(path)
	if err != nil {
		return nil, deviceClass{}
	}

	return dev, class
}

func registerDevice(ctx context.Context, path string, claim *deviceClaim) bool {
	if !claimPath(path, claim) {
		closeHandle(claim.dev)

		return false
	}

	if ctx.Err() != nil {
		unclaimPath(path, claim)
		closeHandle(claim.dev)

		return false
	}

	return true
}

func claimPath(path string, claim *deviceClaim) bool {
	claim.reg.mu.Lock()
	defer claim.reg.mu.Unlock()

	if claim.reg.opened[path] != nil {
		return false
	}

	claim.reg.opened[path] = claim.dev

	return true
}

func unclaimPath(path string, claim *deviceClaim) {
	claim.reg.mu.Lock()
	defer claim.reg.mu.Unlock()

	delete(claim.reg.opened, path)
}

func startReader(ctx context.Context, job *readJob) {
	job.sess.reg.wg.Go(func() {
		readDevice(ctx, job)
	})
}

func tryOpen(ctx context.Context, path string, sess *session) {
	if alreadyOpen(sess.reg, path) {
		return
	}

	dev, class := openUsable(path)
	if dev == nil {
		return
	}

	claim := &deviceClaim{reg: sess.reg, dev: dev}
	if !armDevice(ctx, path, claim) {
		return
	}

	startReader(ctx, &readJob{
		path:  path,
		sess:  sess,
		dev:   dev,
		class: class,
	})
}

func usable(class deviceClass) bool {
	return class.keyboard || class.mouse
}
