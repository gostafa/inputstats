//go:build linux

package linux

import (
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func addInputWatch(deps *hookSet, watchFD int) error {
	watchDesc, watchErr := deps.inotifyAddWatch(
		watchFD,
		inputDir,
		unix.IN_CREATE|unix.IN_ATTRIB|unix.IN_MOVED_TO,
	)
	if watchErr != nil {
		return fmt.Errorf(errFmtWrap, watchErr)
	}

	err := checkWatchDesc(watchDesc)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func checkWatchDesc(watchDesc int) error {
	if watchDesc <= evSyn {
		return errDeviceSkipped
	}

	return nil
}

func closeWatch(deps *hookSet, watchFD int) {
	closeErr := deps.unixClose(watchFD)
	if closeErr != nil {
		return
	}
}

func countInotifyNames(buf []byte) int {
	count := evSyn
	offset := evSyn

	for offset+unixSizeofInotifyEvent <= len(buf) {
		next, name := readInotifyName(buf, offset)
		if next < evSyn {
			break
		}

		offset = next

		count += nameCount(name)
	}

	return count
}

func fillInotifyNames(buf []byte, names []string) {
	index := evSyn
	offset := evSyn

	for offset+unixSizeofInotifyEvent <= len(buf) {
		next, name := readInotifyName(buf, offset)
		if next < evSyn {
			return
		}

		offset = next
		index = storeName(names, index, name)
	}
}

func handleHotplugErr(ctx context.Context, deps *hookSet, err error) bool {
	if !isAgain(err) {
		return false
	}

	return waitAgain(ctx, deps, hotplugPollDelay)
}

func handleHotplugNames(ctx context.Context, sess *session, buf []byte) {
	names := parseInotifyNames(buf)
	for index := range names {
		openHotplugName(ctx, sess, names[index])
	}
}

func initInotify(deps *hookSet) (int, error) {
	watchFD, err := deps.inotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return evSyn, fmt.Errorf(errFmtWrap, err)
	}

	fd, finishErr := finishInotify(deps, watchFD)
	if finishErr != nil {
		return evSyn, fmt.Errorf(errFmtWrap, finishErr)
	}

	return fd, nil
}

func finishInotify(deps *hookSet, watchFD int) (int, error) {
	addErr := addInputWatch(deps, watchFD)
	if addErr != nil {
		closeWatch(deps, watchFD)

		return evSyn, fmt.Errorf(errFmtWrap, addErr)
	}

	return watchFD, nil
}

func nameCount(name string) int {
	if name == emptyName {
		return evSyn
	}

	return int(evKey)
}

func openHotplugName(ctx context.Context, sess *session, name string) {
	if !strings.HasPrefix(name, eventNamePrefix) {
		return
	}

	path := filepath.Join(inputDir, name)
	tryOpen(ctx, path, sess)
	scheduleRetry(ctx, sess, path)
}

func parseInotifyNames(buf []byte) []string {
	count := countInotifyNames(buf)
	if count == evSyn {
		return nil
	}

	names := make([]string, evSyn, count)

	names = names[:cap(names)]
	fillInotifyNames(buf, names)

	return names
}

func pumpHotplug(ctx context.Context, sess *session, watchFD int) {
	buf := make([]byte, evSyn, inotifyBufCap)

	buf = buf[:cap(buf)]

	for {
		if ctx.Err() != nil {
			return
		}

		if !readHotplug(ctx, &hotplugRead{sess: sess, buf: buf, watchFD: watchFD}) {
			return
		}
	}
}

func readHotplug(ctx context.Context, job *hotplugRead) bool {
	nbytes, err := job.sess.deps.unixRead(job.watchFD, job.buf)
	if err != nil {
		return handleHotplugErr(ctx, job.sess.deps, err)
	}

	handleHotplugNames(ctx, job.sess, job.buf[:nbytes])

	return true
}

func readInotifyName(buf []byte, offset int) (next int, name string) {
	end := offset + inotifyLenOff + inotifyLenSize
	nameLen := int(binary.LittleEndian.Uint32(buf[offset+inotifyLenOff : end]))

	offset += unixSizeofInotifyEvent

	if nameLen <= evSyn || offset+nameLen > len(buf) {
		return inotifyStop, emptyName
	}

	name = strings.TrimRight(string(buf[offset:offset+nameLen]), nullByte)

	return offset + nameLen, name
}

func scheduleRetry(ctx context.Context, sess *session, path string) {
	if ctx.Err() != nil {
		return
	}

	sess.deps.afterFunc(hotplugRetryDelay, func() {
		tryOpen(ctx, path, sess)
	})
}

func storeName(names []string, index int, name string) int {
	if name == emptyName {
		return index
	}

	names[index] = name

	return index + int(evKey)
}

func watchHotplug(ctx context.Context, sess *session) {
	watchFD, err := initInotify(sess.deps)
	if err != nil {
		<-ctx.Done()

		return
	}

	defer closeWatch(sess.deps, watchFD)

	pumpHotplug(ctx, sess, watchFD)
}
