package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const app string = "git"

var CWD string

type BranchResponse struct {
	SelectedBranch string    `json:"selectedBranch"`
	Branches       *[]string `json:"branches"`
}

func Branches() (BranchResponse, error) {
	var rtn BranchResponse
	cmd := exec.Command("git", "branch")
	cmd.Dir = CWD
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("%v Error: %v\n", cmd.Args, err)
		return rtn, err
	}
	branches := []string{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line[:2] == "* " {
			branch := strings.TrimSpace(strings.Replace(line, "* ", "", -1))
			rtn.SelectedBranch = branch
			branches = append(branches, branch)
		} else {
			branches = append(branches, strings.TrimSpace(strings.Replace(line, "* ", "", -1)))
		}
	}
	rtn.Branches = &branches
	return rtn, nil
}

type DiffResponse struct {
	Filename     string `json:"filename"`
	LinesChanged int    `json:"linesChanged"`
}

func Diff(branch string) (*[]DiffResponse, error) {
	args := []string{"diff", "--stat=10000", branch}
	out, err := execGitCommand(args)
	if err != nil {
		return nil, err
	}
	response := []DiffResponse{}
	if trimmedOut := strings.TrimSpace(*out); len(trimmedOut) > 0 {
		lines := strings.Split(trimmedOut, "\n")
		for _, line := range lines {
			i := strings.Index(line, "|")
			if i >= 0 {
				filename := strings.TrimSpace(line[:i])
				countStr := strings.TrimSpace(line[i+1:])
				count, err := strconv.Atoi(countStr[:strings.Index(countStr, " ")])
				if err != nil {
					fmt.Printf("Error converting int from %v: %v", countStr[:strings.Index(countStr, " ")], err)
				}
				response = append(response, DiffResponse{Filename: filename, LinesChanged: count})
			}
		}
	}
	return &response, nil
}

const GIT_DATE_LAYOUT string = "Mon Jan 02 15:04:05 2006 -0700"

type LogResponse struct {
	Author struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	Commit   string     `json:"commit"`
	Date     *time.Time `json:"date"`
	Merge    []string   `json:"merge,omitempty"`
	Message  string     `json:"message"`
	Branches []string   `json:"branches,omitempty"`
}

func Log(limit int, branch string) (*[]LogResponse, error) {
	args := []string{"log", "-n", fmt.Sprint(limit), "--decorate=short"}
	if len(branch) > 0 {
		args = append(args, branch)
	}
	out, err := execGitCommand(args)
	if err != nil {
		return nil, err
	}
	response := []LogResponse{}
	if trimmedOut := strings.TrimSpace(*out); len(trimmedOut) > 0 {
		lines := strings.Split(trimmedOut, "\n")
		var logItem *LogResponse
		for _, line := range lines {
			if len(line) > 0 {
				if len(line) > len("commit ") && line[:len("commit ")] == "commit " {
					if logItem != nil {
						logItem.Message = strings.TrimSpace(logItem.Message)
						response = append(response, *logItem)
					}
					logItem = &LogResponse{}
					commit := line[len("commit "):]
					if i := strings.Index(commit, " "); i > 0 {
						commit = commit[:i]
					}
					logItem.Commit = commit
					if i := strings.Index(line, "("); i > 0 {
						eIdx := strings.Index(line, ")")
						branches := strings.Split(line[i+1:eIdx], ", ")
						if strings.Contains(branches[0], "HEAD -> ") {
							branches[0] = branches[0][len("HEAD -> "):]
						}
						logItem.Branches = branches
					}
				} else if len(line) > len("Author: ") && line[:len("Author: ")] == "Author: " {
					author := strings.TrimSpace(line[len("Author: "):])
					logItem.Author.Name = author[:strings.Index(author, "<")]
					logItem.Author.Email = author[strings.Index(author, "<")+1 : strings.Index(author, ">")]
				} else if len(line) > len("Date: ") && line[:len("Date: ")] == "Date: " {
					date := strings.TrimSpace(line[len("Date: "):])
					t, _ := time.Parse(GIT_DATE_LAYOUT, date)
					logItem.Date = &t
				} else if len(line) > len("Merge: ") && line[:len("Merge: ")] == "Merge: " {
					merge := strings.TrimSpace(line[len("Merge: "):])
					logItem.Merge = strings.Split(merge, " ")
				} else if len(line) >= 4 && line[:4] == "    " {
					logItem.Message += line[4:] + "\n"
				}
			} else {
				logItem.Message += "\n"
			}
		}
		if logItem != nil {
			logItem.Message = strings.TrimSpace(logItem.Message)
			response = append(response, *logItem)
		}
	}
	return &response, nil
}

type StatusFile struct {
	Status   string `json:"status"`
	Filename string `json:"filename"`
}

type StatusResponse struct {
	SelectedBranch string       `json:"selectedBranch,omitempty"`
	InSync         *bool        `json:"insync,omitempty"`
	State          string       `json:"state"`
	StagedFiles    []StatusFile `json:"stagedFiles,omitempty"`
	UntrackedFiles []StatusFile `json:"untrackedFiles,omitempty"`
}

func Status() (*StatusResponse, error) {
	args := []string{"status"}
	out, err := execGitCommand(args)
	if err != nil {
		return nil, err
	}
	response := StatusResponse{}
	if trimmedOut := strings.TrimSpace(*out); len(trimmedOut) > 0 {
		i := strings.Index(trimmedOut, "Changes to be committed:")
		if i > 0 {
			response.State = trimmedOut[:i]
			i2 := strings.Index(trimmedOut, "Unmerged paths:")
			if i2 > 0 {
				files := strings.Split(trimmedOut[i:i2], "\n")
				for _, f := range files {
					if len(f) > 1 && f[:1] == "\t" {
						split := strings.Split(f, ":")
						response.StagedFiles = append(
							response.StagedFiles,
							StatusFile{Status: strings.TrimSpace(split[0]), Filename: strings.TrimSpace(split[1])},
						)
					}
				}
				files = strings.Split(trimmedOut[i2:], "\n")
				for _, f := range files {
					if len(f) > 1 && f[:1] == "\t" {
						split := strings.Split(f, ":")
						response.UntrackedFiles = append(
							response.UntrackedFiles,
							StatusFile{Status: strings.TrimSpace(split[0]), Filename: strings.TrimSpace(split[1])},
						)
					}
				}
			}
		} else {
			response.State = trimmedOut
		}
		//check state for branch info
		if strings.Contains(response.State, "On branch ") {
			i := strings.Index(response.State, "On branch ")
			response.SelectedBranch = response.State[i+len("On branch "):]
			response.SelectedBranch = response.SelectedBranch[:strings.Index(response.SelectedBranch, "\n")]
		}
		if strings.Contains(response.State, "have diverged,") {
			sync := false
			response.InSync = &sync
		}
		if strings.Contains(response.State, "Your branch is up to date with ") {
			sync := true
			response.InSync = &sync
		}
	}
	return &response, nil
}

// func Checkout(branch string) (*[]string, error) {
// 	cmd := exec.Command("git", "checkout", branch)
// 	cmd.Dir = CWD
// 	out, err := cmd.Output()
// 	if err != nil {
// 		fmt.Printf("%v Error: %v\n", cmd.Args, err)
// 		return nil, err
// 	}
// 	rtn := []string{}
// 	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
// 	for _, line := range lines {
// 		rtn = append(rtn, strings.TrimSpace(strings.Replace(line, "* ", "", -1)))
// 	}
// 	return &rtn, nil
// }

//long running commands
func execGitCommand(args []string) (*string, error) {
	cmd := exec.Command(app, args...)
	cmd.Dir = CWD
	out, err := cmd.Output()
	if err != nil {
		errStr := fmt.Sprintf("Command error %v %v", cmd.Args, err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			errStr += fmt.Sprintf(":\n%v\n", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf(errStr)
	}
	rtnStr := string(out)
	return &rtnStr, nil
}

func Pull(rebase bool) (*string, error) {
	args := []string{"pull"}
	if rebase {
		args = append(args, "--rebase")
	}
	return execGitCommand(args)
}

func Add(filenames []string) (*string, error) {
	args := []string{"add"}
	args = append(args, filenames...)
	return execGitCommand(args)
}

func Fetch() (*string, error) {
	args := []string{"fetch"}
	return execGitCommand(args)
}

func Push(force bool) (*string, error) {
	args := []string{"push"}
	if force {
		args = append(args, "-f")
	}
	return execGitCommand(args)
}

func Checkout(branch string, filename string) (*string, error) {
	args := []string{"checkout", branch}
	if len(filename) > 0 {
		args = append(args, filename)
	}
	return execGitCommand(args)
}

func DeleteBranch(branch string) (*string, error) {
	args := []string{"branch", "-D", branch}
	return execGitCommand(args)
}

func MoveBranchHead(branch string) (*string, error) {
	args := []string{"branch", "-f", branch}
	return execGitCommand(args)
}

func CreateBranch(branch string) (*string, error) {
	args := []string{"checkout", "-b", branch}
	return execGitCommand(args)
}

func Commit(message string, ammend bool) (*string, error) {
	args := []string{"commit", "-m", message}
	if ammend {
		args = append(args, "--amend")
	}
	return execGitCommand(args)
}

func Rebase(branch string) (*string, error) {
	args := []string{"rebase", branch}
	return execGitCommand(args)
}

func RebaseAbort() (*string, error) {
	args := []string{"rebase", "--abort"}
	return execGitCommand(args)
}
