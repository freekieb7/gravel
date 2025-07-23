package schedule

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

type Scheduler struct {
	jobs []*Job
	mu   sync.RWMutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		jobs: make([]*Job, 0),
	}
}

func (scheduler *Scheduler) AddJob(job *Job) {
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()

	if err := scheduler.validateJob(job); err != nil {
		panic(fmt.Sprintf("invalid job: %v", err))
	}

	scheduler.jobs = append(scheduler.jobs, job)
}

func (scheduler *Scheduler) validateJob(job *Job) error {
	if job.interval <= 0 {
		return fmt.Errorf("job interval must be greater than 0")
	}
	if len(job.tasks) == 0 {
		return fmt.Errorf("job must have at least one task")
	}
	if job.nextExecuteAt.IsZero() {
		job.nextExecuteAt = time.Now().Add(job.interval)
	}
	return nil
}

type Job struct {
	tasks             []Task
	interval          time.Duration
	nextExecuteAt     time.Time
	previousExecuteAt time.Time
	name              string        // For logging/debugging
	maxRetries        int           // Retry failed tasks
	timeout           time.Duration // Task timeout
	mu                sync.RWMutex  // Protect job state
}

func NewJob() *Job {
	return &Job{
		tasks: make([]Task, 0),
	}
}

func (job *Job) WithTasks(tasks ...Task) *Job {
	job.tasks = tasks
	return job
}

func (job *Job) WithInterval(interval time.Duration) *Job {
	job.interval = interval
	return job
}

func (job *Job) WithExecuteAt(executeAt time.Time) *Job {
	job.nextExecuteAt = executeAt
	return job
}

func (job *Job) AddTask(task Task) {
	job.tasks = append(job.tasks, task)
}

func (job *Job) WithName(name string) *Job {
	job.name = name
	return job
}

func (job *Job) WithTimeout(timeout time.Duration) *Job {
	job.timeout = timeout
	return job
}

func (job *Job) WithRetries(maxRetries int) *Job {
	job.maxRetries = maxRetries
	return job
}

type Task struct {
	Function  any
	Arguments []any
}

func NewTask(function any, arguments ...any) (*Task, error) {
	funcValue := reflect.ValueOf(function)
	if funcValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	funcType := funcValue.Type()
	expectedArgs := funcType.NumIn()

	if len(arguments) != expectedArgs {
		return nil, fmt.Errorf("expected %d arguments, got %d", expectedArgs, len(arguments))
	}

	// Validate argument types
	for i, arg := range arguments {
		argType := reflect.TypeOf(arg)
		expectedType := funcType.In(i)
		if !argType.AssignableTo(expectedType) {
			return nil, fmt.Errorf("argument %d: cannot assign %v to %v", i, argType, expectedType)
		}
	}

	return &Task{
		Function:  function,
		Arguments: arguments,
	}, nil
}

func (scheduler *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			{
				scheduler.mu.RLock()
				jobs := make([]*Job, len(scheduler.jobs))
				copy(jobs, scheduler.jobs)
				scheduler.mu.RUnlock()

				now := time.Now()
				for _, job := range jobs {
					if job.shouldExecute(now) {
						go scheduler.executeJob(job, now)
					}
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (job *Job) shouldExecute(now time.Time) bool {
	job.mu.RLock()
	defer job.mu.RUnlock()
	return !job.nextExecuteAt.After(now)
}

func (job *Job) updateNextExecution(now time.Time) {
	job.mu.Lock()
	defer job.mu.Unlock()
	job.previousExecuteAt = now
	job.nextExecuteAt = now.Add(job.interval)
}

func (scheduler *Scheduler) executeJob(job *Job, now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job execution panic: %v", r)
		}
	}()

	for _, task := range job.tasks {
		if err := scheduler.executeTask(task, job.timeout); err != nil {
			log.Printf("Task execution failed: %v", err)
			// Decide: continue with next task or abort job?
		}
	}

	job.updateNextExecution(now)
}

func (scheduler *Scheduler) executeTask(task Task, timeout time.Duration) error {
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- scheduler.doExecuteTask(task)
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return fmt.Errorf("task execution timeout after %v", timeout)
		}
	}

	return scheduler.doExecuteTask(task)
}

func (scheduler *Scheduler) doExecuteTask(task Task) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Task panic: %v", r)
		}
	}()

	function := reflect.ValueOf(task.Function)
	if function.Kind() != reflect.Func {
		return fmt.Errorf("task function is not a function")
	}

	arguments := make([]reflect.Value, len(task.Arguments))
	for k, param := range task.Arguments {
		arguments[k] = reflect.ValueOf(param)
	}

	results := function.Call(arguments)

	// Handle function return values (especially errors)
	if len(results) > 0 {
		if err, ok := results[len(results)-1].Interface().(error); ok && err != nil {
			return err
		}
	}

	return nil
}
