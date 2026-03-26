package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/viktorfroberg/koll/internal/types"
)

func GetChanges(repoPath string) ([]types.FileChange, error) {
	fileMap := make(map[string]*types.FileChange)

	// Unstaged changes
	unstaged, err := runGit(repoPath, "diff", "--name-status")
	if err != nil {
		return nil, err
	}
	parseNameStatus(unstaged, fileMap, false)

	// Staged changes
	staged, err := runGit(repoPath, "diff", "--cached", "--name-status")
	if err != nil {
		return nil, err
	}
	parseNameStatus(staged, fileMap, true)

	// Untracked files
	untracked, err := runGit(repoPath, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(untracked, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if f, ok := fileMap[line]; ok {
			f.Unstaged = true
		} else {
			fileMap[line] = &types.FileChange{
				Path:     line,
				Status:   types.Untracked,
				Unstaged: true,
			}
		}
	}

	// Convert to sorted slice
	files := make([]types.FileChange, 0, len(fileMap))
	for _, f := range fileMap {
		files = append(files, *f)
	}

	// Sort by path
	sortFiles(files)
	return files, nil
}

func GetFileDiff(repoPath string, filePath string, staged bool) ([]types.DiffLine, error) {
	var output string
	var err error

	if staged {
		output, err = runGit(repoPath, "diff", "--cached", "--", filePath)
	} else {
		output, err = runGit(repoPath, "diff", "--", filePath)
	}
	if err != nil {
		return nil, err
	}

	if output == "" {
		// Untracked or new file — read contents directly and show as additions
		return readFileAsAdded(repoPath, filePath)
	}

	return parseDiff(output), nil
}

func readFileAsAdded(repoPath string, filePath string) ([]types.DiffLine, error) {
	fullPath := repoPath + "/" + filePath
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, nil
	}

	// Skip binary files
	if isBinary(content) {
		return []types.DiffLine{
			{Content: "(binary file)", Type: types.Context},
		}, nil
	}

	fileLines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	result := make([]types.DiffLine, 0, len(fileLines)+1)
	result = append(result, types.DiffLine{
		Content: fmt.Sprintf("@@ new file · %d lines @@", len(fileLines)),
		Type:    types.Header,
	})
	for _, line := range fileLines {
		result = append(result, types.DiffLine{
			Content: "+" + line,
			Type:    types.LineAdded,
		})
	}
	return result, nil
}

func isBinary(data []byte) bool {
	// Check first 512 bytes for null bytes
	check := data
	if len(check) > 512 {
		check = check[:512]
	}
	for _, b := range check {
		if b == 0 {
			return true
		}
	}
	return false
}

func parseDiff(output string) []types.DiffLine {
	lines := strings.Split(output, "\n")
	var result []types.DiffLine

	inHeader := true
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			inHeader = false
			result = append(result, types.DiffLine{Content: line, Type: types.Header})
			continue
		}
		if inHeader {
			continue
		}
		if strings.HasPrefix(line, "+") {
			result = append(result, types.DiffLine{Content: line, Type: types.LineAdded})
		} else if strings.HasPrefix(line, "-") {
			result = append(result, types.DiffLine{Content: line, Type: types.LineRemoved})
		} else {
			result = append(result, types.DiffLine{Content: line, Type: types.Context})
		}
	}
	return result
}

func parseNameStatus(output string, fileMap map[string]*types.FileChange, staged bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		statusChar := parts[0]
		path := parts[1]

		// Handle renames: R100\toldpath\tnewpath
		if strings.HasPrefix(statusChar, "R") || strings.HasPrefix(statusChar, "C") {
			pathParts := strings.SplitN(path, "\t", 2)
			if len(pathParts) == 2 {
				path = pathParts[1] // use the new path
			}
		}

		status := charToStatus(statusChar)

		if f, ok := fileMap[path]; ok {
			if staged {
				f.Staged = true
			} else {
				f.Unstaged = true
			}
		} else {
			fileMap[path] = &types.FileChange{
				Path:     path,
				Status:   status,
				Staged:   staged,
				Unstaged: !staged,
			}
		}
	}
}

func charToStatus(s string) types.ChangeStatus {
	if len(s) == 0 {
		return types.Modified
	}
	switch s[0] {
	case 'M':
		return types.Modified
	case 'A':
		return types.Added
	case 'D':
		return types.Deleted
	case 'R':
		return types.Renamed
	case 'C':
		return types.Copied
	default:
		return types.Modified
	}
}

func sortFiles(files []types.FileChange) {
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].Path > files[j].Path {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}

func runGit(repoPath string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func runGitRaw(repoPath string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}
