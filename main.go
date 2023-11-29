package main

import (
	"fmt"
	"git-ui/controller"
	"git-ui/git"
	"git-ui/state"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var cwd string

func main() {
	cwd = os.Getenv("GIT_UI_DIR")
	git.CWD = cwd

	//start controller process
	go controller.ProcessQueue()

	router := gin.Default()
	router.POST("/api/add", postAdd)

	router.GET("/api/branch", getBranches)
	router.POST("/api/branch", postBranch)
	router.DELETE("/api/branch", deleteBranch)
	router.POST("/api/branch/move", postBranchMoveHead)

	router.POST("/api/checkout", postCheckout)

	router.POST("/api/commit", postCommit)

	router.POST("/api/diff", postDiff)

	router.POST("/api/fetch", postFetch)

	router.GET("/api/log", getLog)

	router.POST("/api/pull", postPull)

	router.POST("/api/push", postPush)

	router.POST("/api/rebase", postRebase)
	router.POST("/api/rebase/abort", postRebaseAbort)

	router.GET("/api/state", getState)
	router.GET("/api/state/:id", getItem)
	router.POST("/api/state/:id/restart", postRestart)
	router.POST("/api/state/:id/void", postVoid)

	router.GET("/api/status", getStatus)

	router.GET("/health", healthCheck)

	router.Static("/html", "./html")

	router.Run("localhost:8111")
}

func healthCheck(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, time.Now().Format(time.RFC3339))
}

func getBranches(c *gin.Context) {
	if state.CanExecute() {
		result, err := git.Branches()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error"})
			return
		}
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"message": "state not ready for execution, check /api/state endpoint"})
	}
}

func getStatus(c *gin.Context) {
	if state.CanExecute() {
		result, err := git.Status()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error"})
			return
		}
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"message": "state not ready for execution, check /api/state endpoint"})
	}
}

func postBranch(c *gin.Context) {
	var body checkoutPost
	if err := c.BindJSON(&body); err != nil {
		return
	}
	response := createStateItem(
		"Create branch "+body.Branch,
		func() (*string, error) { return git.CreateBranch(body.Branch) },
	)
	c.JSON(http.StatusOK, response)
}

func postBranchMoveHead(c *gin.Context) {
	var body checkoutPost
	if err := c.BindJSON(&body); err != nil {
		return
	}
	response := createStateItem(
		"Move branch head "+body.Branch,
		func() (*string, error) { return git.MoveBranchHead(body.Branch) },
	)
	c.JSON(http.StatusOK, response)
}

func deleteBranch(c *gin.Context) {
	var body checkoutPost
	if err := c.BindJSON(&body); err != nil {
		return
	}
	response := createStateItem(
		"Delete branch "+body.Branch,
		func() (*string, error) { return git.DeleteBranch(body.Branch) },
	)
	c.JSON(http.StatusOK, response)
}

func postDiff(c *gin.Context) {
	if state.CanExecute() {
		var body checkoutPost
		if err := c.BindJSON(&body); err != nil {
			return
		}
		result, err := git.Diff(body.Branch)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error"})
			return
		}
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"message": "state not ready for execution, check /api/state endpoint"})
	}
}

func getLog(c *gin.Context) {
	if state.CanExecute() {
		limitStr := c.Query("limit")
		limit, e := strconv.Atoi(limitStr)
		if e != nil {
			limit = 25
		}

		branch := c.Query("branch")

		result, err := git.Log(limit, branch)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error"})
			return
		}
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"message": "state not ready for execution, check /api/state endpoint"})
	}
}

func getState(c *gin.Context) {
	var sinceUpdate *time.Time
	lastUpdateStr := c.Query("lastUpdate")
	if len(lastUpdateStr) > 0 {
		milsec, err := strconv.ParseInt(lastUpdateStr, 10, 64)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error"})
			return
		}
		t := time.Unix(0, milsec*int64(time.Millisecond))
		sinceUpdate = &t
	}

	//c.Request().Context().Done()
	result := state.GetState(sinceUpdate)
	c.JSON(http.StatusOK, result)
}

func getItem(c *gin.Context) {
	id := c.Param("id")
	item, err := state.GetItem(id)
	if err != nil {
		errStr := fmt.Sprintf("Error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": errStr})
		return
	}
	c.JSON(http.StatusOK, item)
}

func postRestart(c *gin.Context) {
	id := c.Param("id")
	err := state.UpdateStatus(id, state.STATUS_RESTART, nil, nil)
	if err != nil {
		errStr := fmt.Sprintf("Error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": errStr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Restarted"})
}

func postVoid(c *gin.Context) {
	id := c.Param("id")
	err := state.UpdateStatus(id, state.STATUS_VOID, nil, nil)
	if err != nil {
		errStr := fmt.Sprintf("Error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": errStr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Voided"})
}

type checkoutPost struct {
	Branch string `json:"branch"`
	File   string `json:"file,omitempty"`
}

func postCheckout(c *gin.Context) {
	var body checkoutPost
	if err := c.BindJSON(&body); err != nil {
		return
	}

	response := createStateItem(
		"Checkout "+body.Branch,
		func() (*string, error) { return git.Checkout(body.Branch, body.File) },
	)
	c.JSON(http.StatusOK, response)
}

func postRebase(c *gin.Context) {
	var body checkoutPost
	if err := c.BindJSON(&body); err != nil {
		return
	}

	response := createStateItem(
		"Rebase "+body.Branch,
		func() (*string, error) { return git.Rebase(body.Branch) },
	)
	c.JSON(http.StatusOK, response)
}

func postRebaseAbort(c *gin.Context) {
	response := createStateItem(
		"Rebase abort",
		func() (*string, error) { return git.RebaseAbort() },
	)
	c.JSON(http.StatusOK, response)
}

type commitPost struct {
	Amend   bool   `json:"amend"`
	Message string `json:"message"`
}

func postCommit(c *gin.Context) {
	var body commitPost
	if err := c.BindJSON(&body); err != nil {
		return
	}

	response := createStateItem(
		"Commit",
		func() (*string, error) { return git.Commit(body.Message, body.Amend) },
	)
	c.JSON(http.StatusOK, response)
}

func postPull(c *gin.Context) {
	response := createStateItem(
		"Pull",
		func() (*string, error) { return git.Pull(true) },
	)
	c.JSON(http.StatusOK, response)
}

func postPush(c *gin.Context) {
	f := c.Query("f")
	force := f == "true"
	response := createStateItem(
		"Push",
		func() (*string, error) { return git.Push(force) },
	)
	c.JSON(http.StatusOK, response)
}

func postFetch(c *gin.Context) {
	response := createStateItem(
		"Fetch",
		func() (*string, error) { return git.Fetch() },
	)
	c.JSON(http.StatusOK, response)
}

type addPost struct {
	Files []string `json:"files"`
}

func postAdd(c *gin.Context) {
	var body addPost
	if err := c.BindJSON(&body); err != nil {
		return
	}

	response := createStateItem(
		"Add",
		func() (*string, error) { return git.Add(body.Files) },
	)
	c.JSON(http.StatusOK, response)
}

type createStateResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
	Href    string `json:"href"`
}

func createStateItem(name string, fn func() (*string, error)) createStateResponse {
	id := state.AddItem(
		name,
		state.STATUS_CREATING,
		fn,
	)
	return createStateResponse{Message: "Created " + name, ID: id, Href: "/api/state/" + id}
}
