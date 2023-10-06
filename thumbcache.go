package main

import (
	"io"
	"net/http"
	"os"
	"path"
)

func RenderThumbCache(w http.ResponseWriter, r *http.Request, hash string, file string) bool {
	if !config.CacheThumbnails {
		return false
	}

	w.Header().Set("Content-Type", "image/jpeg")
	SetCacheHeader(w, r, 31536000)

	fname := config.CacheFile("thing/file", hash, file)
	os.MkdirAll(path.Dir(fname), 0755)

	f, err := os.Open(fname)

	if os.IsNotExist(err) {
		debugPrintf("Cache miss: %s\n", fname)
		return false
	}

	if err != nil {
		debugPrintf("Cache read error: %s (%s)\n", fname, err)
		return false
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	if err != nil {
		debugPrintf("Cache send error: %s (%s)\n", fname, err)
		return false

	}

	debugPrintf("Cache hit: %s\n", fname)
	return true
}

func ThumbCacheTarget(hash string, file string) io.WriteCloser {
	fname := config.CacheFile("thing/file", hash, file)

	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		debugPrintf("Cache write error: %s (%s)\n", fname, err)
		return nil
	}

	debugPrintf("Cached: %s\n", fname)
	return f
}
