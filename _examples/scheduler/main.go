package main

import (
	"context"
	"log"
	"time"

	"github.com/freekieb7/gravel/schedule"
)

func main() {
	scheduler := schedule.NewScheduler()

	// Create a job with error handling
	job := schedule.NewJob().
		WithName("daily-cleanup").
		WithInterval(24 * time.Hour).
		WithTimeout(30 * time.Minute).
		WithRetries(3).
		WithExecuteAt(time.Now().Add(time.Hour))

	// Add tasks with validation
	cleanupTask, err := schedule.NewTask(cleanupFiles, "/tmp")
	if err != nil {
		log.Fatal(err)
	}
	job.AddTask(*cleanupTask)

	scheduler.AddJob(job)

	// Run with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Run(ctx); err != nil {
		log.Printf("Scheduler stopped: %v", err)
	}
}

func cleanupFiles(path string) error {
	log.Printf("Cleaning up files in %s", path)
	// Cleanup logic here
	return nil
}
