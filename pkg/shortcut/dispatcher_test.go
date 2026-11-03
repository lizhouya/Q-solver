package shortcut

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestSerialActionDispatcherEnqueueDoesNotWaitForBlockedHandlerAndKeepsFIFO(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	completed := make(chan string, 3)
	var once sync.Once

	d := newSerialActionDispatcher(func(action string) {
		if action == "screenshot-1" {
			once.Do(func() { close(firstStarted) })
			<-releaseFirst
		}
		completed <- action
	})
	d.Start()
	defer d.Stop()

	if !d.Enqueue("screenshot-1") {
		t.Fatal("首个动作应成功入队")
	}
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("首个动作未开始执行")
	}

	enqueueDone := make(chan bool, 1)
	go func() {
		enqueueDone <- d.Enqueue("screenshot-2") && d.Enqueue("send")
	}()
	select {
	case ok := <-enqueueDone:
		if !ok {
			t.Fatal("处理器阻塞期间后续动作仍应成功入队")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("入队被正在执行的耗时动作阻塞")
	}

	close(releaseFirst)
	got := make([]string, 0, 3)
	for len(got) < 3 {
		select {
		case action := <-completed:
			got = append(got, action)
		case <-time.After(time.Second):
			t.Fatalf("等待动作完成超时，已收到 %v", got)
		}
	}
	want := []string{"screenshot-1", "screenshot-2", "send"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("动作必须保持 FIFO：want %v, got %v", want, got)
	}
}

func TestSerialActionDispatcherRunsOnlyOneHandlerAtATime(t *testing.T) {
	var mu sync.Mutex
	active := 0
	maxActive := 0
	completed := make(chan struct{}, 4)

	d := newSerialActionDispatcher(func(string) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()
		completed <- struct{}{}
	})
	d.Start()
	defer d.Stop()

	for _, action := range []string{"one", "two", "three", "four"} {
		if !d.Enqueue(action) {
			t.Fatalf("动作 %q 应成功入队", action)
		}
	}
	for i := 0; i < 4; i++ {
		select {
		case <-completed:
		case <-time.After(time.Second):
			t.Fatal("等待串行动作完成超时")
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if maxActive != 1 {
		t.Fatalf("处理器不得并发执行，最大并发数为 %d", maxActive)
	}
}

func TestSerialActionDispatcherRejectsActionsAfterStop(t *testing.T) {
	d := newSerialActionDispatcher(func(string) {})
	d.Start()
	d.Stop()
	d.Stop()

	if d.Enqueue("late") {
		t.Fatal("停止后不得接受新动作")
	}
}
