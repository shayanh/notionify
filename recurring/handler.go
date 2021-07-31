package recurring

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type TasksHandler struct {
	nh  *NotionHandler
	log *logrus.Logger
}

func NewTasksHandler(nh *NotionHandler, log *logrus.Logger) *TasksHandler {
	return &TasksHandler{
		nh:  nh,
		log: log,
	}
}

func dateEquals(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func (th *TasksHandler) Handle(ctx context.Context) error {
	tasks, err := th.nh.ListTasks(ctx)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if task.Status != statusDone || (!task.DueDate.IsZero() && dateEquals(task.DueDate, time.Now())) {
			continue
		}
		updatedTask := &NotionTask{
			ID:      task.ID,
			Status:  "",
			DueDate: time.Now(),
		}
		retTask, err := th.nh.UpdateTask(ctx, updatedTask)
		if err != nil {
			return err
		}
		th.log.WithFields(logrus.Fields{
			"ID":       retTask.ID,
			"Name":     retTask.Name,
			"Due Date": retTask.DueDate,
		}).Info("notion task updated")
	}
	return nil
}
