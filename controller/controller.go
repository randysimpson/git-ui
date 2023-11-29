package controller

import (
	"fmt"
	"git-ui/state"
	"time"
)

func ProcessQueue() {
	for {
		i := state.GetNextItem()
		if i != nil {
			//process
			err := state.UpdateStatus(i.ID, state.STATUS_EXECUTING, nil, nil)
			if err != nil {
				fmt.Printf("Status Error: %v\n", err)
			} else {
				//only continue if we were able to set status successfully
				out, err := i.Method()
				if err != nil {
					errStr := fmt.Sprintf("Execution Error: %v", err)
					//fmt.Printf("%s\n", errStr)
					err = state.UpdateStatus(i.ID, state.STATUS_ERROR, nil, &errStr)
					if err != nil {
						fmt.Printf("Status Error: %v\n", err)
					}
				} else {
					err = state.UpdateStatus(i.ID, state.STATUS_COMPLETE, out, nil)
					if err != nil {
						fmt.Printf("Status Error: %v\n", err)
					}
				}
			}
		} else {
			//wait 5 sec
			time.Sleep(5 * time.Second)
		}
	}
}
