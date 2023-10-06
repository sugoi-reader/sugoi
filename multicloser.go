package main

import "io"

type MultiCloser []io.Closer

func MultiClose(closers MultiCloser) {
	for _, closer := range closers {
		if closer != nil {
			closer.Close()
		}
	}
}
