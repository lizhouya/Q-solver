package shortcut

import "sync"

type serialActionDispatcher struct {
	mu       sync.Mutex
	cond     *sync.Cond
	queue    []string
	handler  func(string)
	started  bool
	stopped  bool
	done     chan struct{}
	doneOnce sync.Once
}

func newSerialActionDispatcher(handler func(string)) *serialActionDispatcher {
	d := &serialActionDispatcher{
		handler: handler,
		done:    make(chan struct{}),
	}
	d.cond = sync.NewCond(&d.mu)
	return d
}

func (d *serialActionDispatcher) Start() {
	d.mu.Lock()
	if d.started || d.stopped {
		d.mu.Unlock()
		return
	}
	d.started = true
	d.mu.Unlock()

	go d.run()
}

func (d *serialActionDispatcher) Enqueue(action string) bool {
	if action == "" {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.started || d.stopped {
		return false
	}
	d.queue = append(d.queue, action)
	d.cond.Signal()
	return true
}

func (d *serialActionDispatcher) Stop() {
	d.mu.Lock()
	if !d.stopped {
		d.stopped = true
		d.queue = nil
		d.cond.Broadcast()
	}
	started := d.started
	d.mu.Unlock()

	if !started {
		d.doneOnce.Do(func() { close(d.done) })
	}
	<-d.done
}

func (d *serialActionDispatcher) run() {
	defer d.doneOnce.Do(func() { close(d.done) })

	for {
		d.mu.Lock()
		for len(d.queue) == 0 && !d.stopped {
			d.cond.Wait()
		}
		if d.stopped {
			d.queue = nil
			d.mu.Unlock()
			return
		}
		action := d.queue[0]
		d.queue[0] = ""
		d.queue = d.queue[1:]
		handler := d.handler
		d.mu.Unlock()

		if handler != nil {
			handler(action)
		}
	}
}
