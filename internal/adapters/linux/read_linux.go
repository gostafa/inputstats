//go:build linux

package linux

import (
	"context"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

func closeAll(reg *registry) {
	handles := snapshotHandles(reg)
	for index := range handles {
		closeHandle(handles[index])
	}
}

func closeHandle(dev evdevHandle) {
	closeErr := dev.Close()
	if closeErr != nil {
		return
	}
}

func deliverCombo(ctx context.Context, job *dispatchJob) {
	typ := uint16(job.event.Type)
	code := uint16(job.event.Code)

	kind, ok := parseEvdev(evdevSample{typ: typ, code: code, value: job.event.Value}, job.state)
	if !ok {
		return
	}

	ports.Deliver(ctx, job.events, domain.Event{Type: kind})
}

func deliverSingle(ctx context.Context, job *dispatchJob) {
	if job.class.keyboard {
		handleKeyEvent(ctx, job.events, keySample{
			code:  uint16(job.event.Code),
			value: job.event.Value,
		})

		return
	}

	if job.class.mouse {
		handleMouseFromJob(ctx, job)
	}
}

func dispatch(ctx context.Context, job *dispatchJob) {
	if job.class.keyboard && job.class.mouse {
		deliverCombo(ctx, job)

		return
	}

	deliverSingle(ctx, job)
}

func finishReader(job *readJob) {
	unregister(job.sess.reg, job.path, job.dev)
	closeHandle(job.dev)
}

func handleKeyEvent(ctx context.Context, events chan<- domain.Event, sample keySample) {
	kind, ok := parseKey(sample.code, sample.value)
	if !ok {
		return
	}

	ports.Deliver(ctx, events, domain.Event{Type: kind})
}

func handleMouseFromJob(ctx context.Context, job *dispatchJob) {
	handleMouseEvent(ctx, job.events, mouseSample{
		state: job.state,
		typ:   uint16(job.event.Type),
		code:  uint16(job.event.Code),
		value: job.event.Value,
	})
}

func handleMouseEvent(ctx context.Context, events chan<- domain.Event, sample mouseSample) {
	switch sample.typ {
	case evKey:
		deliverMouseButton(ctx, events, sample)
	case evRel:
		noteRel(sample.code, sample.state)
	case evSyn:
		deliverMouseSyn(ctx, events, sample)
	default:
	}
}

func deliverMouseButton(ctx context.Context, events chan<- domain.Event, sample mouseSample) {
	kind, ok := parseMouseButton(sample.code, sample.value)
	if !ok {
		return
	}

	ports.Deliver(ctx, events, domain.Event{Type: kind})
}

func deliverMouseSyn(ctx context.Context, events chan<- domain.Event, sample mouseSample) {
	kind, ok := flushSyn(sample.code, sample.state)
	if !ok {
		return
	}

	ports.Deliver(ctx, events, domain.Event{Type: kind})
}

func handleReadErr(ctx context.Context, deps *hookSet, err error) bool {
	if !isAgain(err) {
		return false
	}

	return waitAgain(ctx, deps, againDelay)
}

func readDevice(ctx context.Context, job *readJob) {
	defer finishReader(job)

	var state mouseState

	for {
		if ctx.Err() != nil {
			return
		}

		if !readOneEvent(ctx, job, &state) {
			return
		}
	}
}

func readOneEvent(ctx context.Context, job *readJob, state *mouseState) bool {
	event, err := job.dev.ReadOne()
	if err != nil {
		return handleReadErr(ctx, job.sess.deps, err)
	}

	dispatch(ctx, &dispatchJob{
		event:  event,
		class:  job.class,
		state:  state,
		events: job.sess.events,
	})

	return true
}

func snapshotHandles(reg *registry) []evdevHandle {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	handles := make([]evdevHandle, evSyn, len(reg.opened))
	for path := range reg.opened {
		handles = append(handles, reg.opened[path])
	}

	return handles
}

func unregister(reg *registry, path string, dev evdevHandle) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	current := reg.opened[path]
	if current != dev {
		return
	}

	delete(reg.opened, path)
}

func waitAgain(ctx context.Context, deps *hookSet, delay time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-deps.afterDelay(delay):
		return true
	}
}
