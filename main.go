package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "sync"

    "github.com/sirupsen/logrus"
)

type Pool struct {
    Tasks []*Task

    concurrency int
    tasksChan   chan *Task
    wg          sync.WaitGroup
}

func newPool(task []*Task, concurrency int) *Pool {
    return &Pool{
        Tasks:       task,
        concurrency: concurrency,
        tasksChan:   make(chan *Task),
    }
}

func (p *Pool) work() {
    for task := range p.tasksChan {
        task.Run(&p.wg)
    }
}

func (p *Pool) Run() {
    for i := 0; i < p.concurrency; i++ {
        go p.work()
    }
    p.wg.Add(len(p.Tasks))
    for _, task := range p.Tasks {
        p.tasksChan <- task
    }
    close(p.tasksChan)
    p.wg.Wait()
}

type Task struct {
    Err error
    f   func() error
}

func newTask(f func() error) *Task {
    return &Task{f: f}
}

func (t *Task) Run(wg *sync.WaitGroup) {
    t.Err = t.f()
    wg.Done()
}

type fooReturn struct {
    Status  int    `json:"status"`
    Message string `json:"message"`
    Module  string `json:"module"`
}

func main() {
    var mailID = [2]int{1000, 10001}
    fmt.Printf("mail count is %d\n", len(mailID))
    var tasks []*Task
    for _, i := range mailID {
        j := i
        tasks = append(tasks, newTask(func() error {
            url := "aaaaa"
            payload := strings.NewReader(fmt.Sprintf("{\n\t\"relanch_id\": %d\n}", j))
            req, _ := http.NewRequest("POST", url, payload)
            req.Header.Add("content-type", "application/json")
            res, err := http.DefaultClient.Do(req);
            if err != nil {
                return fmt.Errorf("relanch error for %d err is %v", j, err)
            }
            defer res.Body.Close()
            foo := new(fooReturn)
            err = json.NewDecoder(res.Body).Decode(&foo)
            if err != nil {
                return fmt.Errorf("relanch error for %d err is %v", j, err)
            }
            if foo.Status == 2000 {
                logrus.Info("relanch success for ", j)
                return nil
            } else {
                return fmt.Errorf("relanch error for %d status is %d and message is %s", j, foo.Status, foo.Message)
            }
        }))
    }

    p := newPool(tasks, 4)
    p.Run()
    var numErrors int
    for _, task := range p.Tasks {
        if task.Err != nil {
            logrus.Error(task.Err)
            numErrors++
        }
    }
}
