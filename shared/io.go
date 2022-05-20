package shared

import (
	"io"
	"sync"
)

func Bridge(src, dst io.ReadWriteCloser) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		_, _ = io.Copy(src, dst)
		_ = src.Close()
		_ = dst.Close()
		wg.Done()
	}()
	go func() {
		_, _ = io.Copy(dst, src)
		_ = src.Close()
		_ = dst.Close()
		wg.Done()
	}()
	wg.Wait()
}
