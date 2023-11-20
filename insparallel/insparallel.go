package insparallel

import (
	"sync"
)

type Func[T, V any] func(T) (V, error)

type Work[T, V any] struct {
	Params T
	RetVal V
	Err    error
	Done   bool
	Func   Func[T, V]
}

type WorkGroup[T, V any] struct {
	Works               []*Work[T, V]
	MaxConcurrencyCount int
	waitGroup           *sync.WaitGroup
}

type WorkGroups[T, V any] struct {
	WorkGroups *[]*WorkGroup[T, V]
}

var mutex sync.Mutex

func NewWorkGroup[T, V any](maxConcurrencyCount int) *WorkGroup[T, V] {
	return &WorkGroup[T, V]{
		MaxConcurrencyCount: maxConcurrencyCount,
		Works:               []*Work[T, V]{},
		waitGroup:           &sync.WaitGroup{},
	}
}

func (wg *WorkGroup[T, V]) Run() {
	concurrentGoroutines := make(chan struct{}, wg.MaxConcurrencyCount)
	for _, work := range wg.Works {
		wg.waitGroup.Add(1)
		w := work
		go func() {
			defer wg.waitGroup.Done()
			concurrentGoroutines <- struct{}{}
			w.RetVal, w.Err = w.Func(w.Params)
			w.Done = true
			<-concurrentGoroutines
		}()
	}
}

func (wg *WorkGroup[T, V]) Wait() {
	wg.waitGroup.Wait()
}

func (wg *WorkGroup[T, V]) Add(f Func[T, V], params T) {
	work := Work[T, V]{
		Params: params,
		Func:   f,
	}
	wg.Works = append(wg.Works, &work)
}

func NewWorkGroups[T, V any]() WorkGroups[T, V] {
	return WorkGroups[T, V]{
		WorkGroups: &[]*WorkGroup[T, V]{},
	}
}

func (wgs *WorkGroups[T, V]) RunAll() {
	for _, wg := range *wgs.WorkGroups {
		wg.Run()
	}
}

func (wgs *WorkGroups[T, V]) WaitAll() {
	for _, w := range *wgs.WorkGroups {
		w.waitGroup.Wait()
	}
}

func (wgs *WorkGroups[T, V]) Add(wg *WorkGroup[T, V]) {
	*wgs.WorkGroups = append(*wgs.WorkGroups, wg)
}

func WithMutex(f func()) {
	mutex.Lock()
	f()
	mutex.Unlock()
}
