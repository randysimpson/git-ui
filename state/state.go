package state

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type jsonTime struct {
	time.Time
}

func (j *jsonTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + j.UTC().Format("2006-01-02T15:04:05.999Z07:00") + `"`), nil
}

func (self *jsonTime) UnmarshalJSON(b []byte) (err error) {
	s := string(b)

	// Get rid of the quotes "" around the value.
	// A second option would be to include them
	// in the date format string instead, like so below:
	//   time.Parse(`"`+time.RFC3339Nano+`"`, s)
	t, err := time.Parse(`"2006-01-02T15:04:05.999Z07:00"`, s)
	if err != nil {
		return err
	}
	self.Time = t
	return
}

func (self *jsonTime) After(u time.Time) bool {
	//check milliseconds
	return self.UnixMilli() > u.UnixMilli()
}

func (self *jsonTime) Before(u time.Time) bool {
	//check milliseconds
	return self.UnixMilli() < u.UnixMilli()
}

type status struct {
	LastUpdate *jsonTime `json:"lastUpdate,omitempty"`
	Items      []*item   `json:"items"`
}

type executionStats struct {
	Logs      *string   `json:"logs,omitempty"`
	Error     *string   `json:"error,omitempty"`
	StartTime jsonTime  `json:"startTime,omitempty"`
	EndTime   *jsonTime `json:"endTime,omitempty"`
}

type item struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	Status         string                  `json:"status"`
	Created        jsonTime                `json:"created"`
	Modified       jsonTime                `json:"modified"`
	Method         func() (*string, error) `json:"-"`
	ExecutionStats []*executionStats       `json:"executionStats,omitempty"`
}

var currentState = []*item{}
var lastUpdate = jsonTime{}
var ps = NewPubSub()

const STATUS_CREATING string = "Creating"
const STATUS_COMPLETE string = "Complete"
const STATUS_ERROR string = "Error"
const STATUS_EXECUTING string = "Executing"
const STATUS_RESTART string = "Restarting"
const STATUS_VOID string = "Void/Skip"

var idIndexMap = map[string]int{}

func AddItem(name string, status string, fn func() (*string, error)) string {
	id := fmt.Sprintf("%v", uuid.New())
	idIndexMap[id] = len(currentState)
	currentState = append(currentState, &item{ID: id, Name: name, Status: status, Created: jsonTime{time.Now()}, Modified: jsonTime{time.Now()}, Method: fn})
	lastUpdate = jsonTime{time.Now()}
	ps.Publish()
	return id
}

func GetNextItem() *item {
	for i := 0; i < len(currentState); i++ {
		if currentState[i].Status != STATUS_COMPLETE && currentState[i].Status != STATUS_VOID {
			if currentState[i].Status == STATUS_CREATING || currentState[i].Status == STATUS_RESTART {
				return currentState[i]
			} else {
				return nil
			}
		}
	}
	return nil
}

func UpdateStatus(id string, newStatus string, logs *string, err *string) error {
	if i, ok := idIndexMap[id]; ok {
		if newStatus == STATUS_RESTART && currentState[i].Status != STATUS_ERROR {
			//can only restart if error
			return fmt.Errorf(
				"Cannot restart %s because current status is not '%s': %v\n",
				id,
				STATUS_ERROR,
				currentState[i].Status,
			)
		} else if newStatus == STATUS_VOID &&
			(currentState[i].Status != STATUS_ERROR && currentState[i].Status != STATUS_CREATING && currentState[i].Status != STATUS_RESTART) {
			return fmt.Errorf(
				"Cannot void %s because current status is not '%s','%s' or '%s': %v\n",
				id,
				STATUS_ERROR,
				STATUS_CREATING,
				STATUS_RESTART,
				currentState[i].Status,
			)
		}
		currentTime := time.Now()
		currentState[i].Status = newStatus
		currentState[i].Modified = jsonTime{currentTime}
		if newStatus == STATUS_EXECUTING {
			if currentState[i].ExecutionStats == nil {
				currentState[i].ExecutionStats = []*executionStats{}
			}
			currentState[i].ExecutionStats = append(currentState[i].ExecutionStats, &executionStats{StartTime: jsonTime{currentTime}})
		} else if newStatus == STATUS_ERROR || newStatus == STATUS_COMPLETE {
			currentState[i].ExecutionStats[len(currentState[i].ExecutionStats)-1].EndTime = &jsonTime{currentTime}
			currentState[i].ExecutionStats[len(currentState[i].ExecutionStats)-1].Logs = logs
			currentState[i].ExecutionStats[len(currentState[i].ExecutionStats)-1].Error = err
		}
		lastUpdate = jsonTime{currentTime}
		ps.Publish()
		return nil
	} else {
		return fmt.Errorf("No id %s found in current state: %v\n", id, currentState)
	}
}

func CanExecute() bool {
	for i := 0; i < len(currentState); i++ {
		if currentState[i].Status != STATUS_COMPLETE && currentState[i].Status != STATUS_VOID {
			return false
		}
	}
	return true
}

func GetState(updateTime *time.Time) status {
	if updateTime != nil {
		sinceUpdate := jsonTime{*updateTime}
		if sinceUpdate.Before(lastUpdate.Time) {
			rtnItems := []*item{}
			for _, i := range currentState {
				if sinceUpdate.Before(i.Modified.Time) {
					rtnItems = append(rtnItems, i)
				}
			}
			return status{LastUpdate: &lastUpdate, Items: rtnItems}
		} else {
			ch, close := ps.Subscribe()
			defer close()

			select {
			case <-ch:
				return GetState(&sinceUpdate.Time)
				//need to add timemout?
			}
		}
	} else {
		return status{LastUpdate: &lastUpdate, Items: currentState}
	}
}

func GetItem(id string) (*item, error) {
	if i, ok := idIndexMap[id]; ok {
		return currentState[i], nil
	} else {
		return nil, fmt.Errorf("No id %s found in current state: %v\n", id, currentState)
	}
}
