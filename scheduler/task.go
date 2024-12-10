package scheduler

import (
	"context"
	"reflect"
	"time"
)

type Scheduler struct {
	jobs []*Job
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		jobs: make([]*Job, 0),
	}
}

func (scheduler *Scheduler) AddJob(job Job) {
	scheduler.jobs = append(scheduler.jobs, &job)
}

type Job struct {
	tasks             []Task
	interval          time.Duration
	nextExecuteAt     time.Time
	previousExecuteAt time.Time
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

type Task struct {
	Function  any
	Arguments []any
}

func NewTask(function any, arguments ...any) *Task {
	numberOfExpectedArguments := reflect.ValueOf(function).Type().NumIn()
	if len(arguments) != numberOfExpectedArguments {
		panic("number of expected vs provided arguments do not match")
	}

	return &Task{function, arguments}
}

func (scheduler *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			{
				now := time.Now()

				for _, job := range scheduler.jobs {
					if job.nextExecuteAt.After(now) {
						continue
					}

					// Each job is executed asynchronasly
					go func() {
						// Each task is executed sequentialy
						for _, task := range job.tasks {
							arguments := make([]reflect.Value, len(task.Arguments))
							for k, param := range task.Arguments {
								arguments[k] = reflect.ValueOf(param)
							}

							function := reflect.ValueOf(task.Function)
							function.Call(arguments) // Panics when Call is used on a non function
						}
					}()

					job.previousExecuteAt = now
					job.nextExecuteAt = now.Add(job.interval)
				}
			}
		case <-ctx.Done():
			{
				ticker.Stop()
				return
			}
		}
	}
}
