package main

import (
	"fmt"
)

type ReindexJob struct {
	Running       bool
	RequestCancel bool `json:"-"`
	Processed     int
	Total         int
	Error         int
	Ok            int
	Log           []string
}

func (this *ReindexJob) Start() error {
	const BATCH_LIMIT = 500
	if this.Running {
		return fmt.Errorf("Reindex already started")
	}

	this.Running = true
	this.RequestCancel = false
	this.Processed = 0
	this.Total = len(filePointers.List)
	this.Error = 0
	this.Ok = 0
	this.Log = nil

	go func() {
		var err error
		batch := bleveIndex.NewBatch()

		i := 0
		for _, file := range filePointers.List {

			if this.RequestCancel {
				this.Log = append(this.Log, "Cancelled")
				break
			}

			err = file.ReindexIntoBatch(batch)
			if err != nil {
				this.Error++
				this.Processed++
				this.Log = append(this.Log, fmt.Sprintf("%s: %s", file.Key, err.Error()))
			} else {
				i++
				this.Ok++
				this.Processed++
				// this.Log = append(this.Log, fmt.Sprintf("%s: %s", file.Key, "OK"))
			}

			if i >= BATCH_LIMIT {
				this.Log = append(this.Log, fmt.Sprintf("Processing batch of %d files", BATCH_LIMIT))
				i = 0
				err = bleveIndex.Batch(batch)
				if err != nil {
					this.Log = append(this.Log, fmt.Sprintf("Error: %s", err.Error()))
				} else {
					if this.Total != 0 {
						this.Log = append(this.Log, fmt.Sprintf("%.1f%% done", (float64(this.Processed)/float64(this.Total))*100.0))
					}
				}
				batch = bleveIndex.NewBatch()
			}
		}

		if i > 0 {
			this.Log = append(this.Log, fmt.Sprintf("Processing final batch of %d files", i))
			err = bleveIndex.Batch(batch)
			if err != nil {
				this.Log = append(this.Log, fmt.Sprintf("Error: %s", err.Error()))
			} else {
				this.Log = append(this.Log, "100% done!")
			}
		}

		this.Running = false
	}()

	return nil
}
