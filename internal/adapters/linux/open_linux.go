//go:build linux

package linux

import (
	"context"
	"fmt"
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

func openAndClassify(deps *hookSet, path string) (classified, error) {
	dev, err := deps.openDevice(path)
	if err != nil {
		return classified{}, fmt.Errorf(errFmtWrap, err)
	}

	result, classErr := classifyOpened(dev)
	if classErr != nil {
		return classified{class: result.class}, fmt.Errorf(errFmtWrap, classErr)
	}

	return result, nil
}

func classifyOpened(dev evdevDevice) (classified, error) {
	class := classifyDevice(dev)
	if !usable(class) {
		closeHandle(dev)

		return classified{class: class}, errDeviceSkipped
	}

	return classified{dev: dev, class: class}, nil
}

func openUsable(deps *hookSet, path string) classified {
	result, err := openAndClassify(deps, path)
	if err != nil {
		return classified{}
	}

	return result
}

func registerDevice(ctx context.Context, path string, claim *deviceClaim) bool {
	if !claimPath(path, claim) {
		closeHandle(claim.dev)

		return false
	}

	return keepClaim(ctx, path, claim)
}

func keepClaim(ctx context.Context, path string, claim *deviceClaim) bool {
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

	result := openUsable(sess.deps, path)
	if result.dev == nil {
		return
	}

	claim := &deviceClaim{reg: sess.reg, dev: result.dev}
	if !armDevice(ctx, path, claim) {
		return
	}

	startReader(ctx, &readJob{
		path:  path,
		sess:  sess,
		dev:   result.dev,
		class: result.class,
	})
}

func usable(class deviceClass) bool {
	return class.keyboard || class.mouse
}
