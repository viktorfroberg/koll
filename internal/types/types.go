package types

type ChangeStatus int

const (
	Modified ChangeStatus = iota
	Added
	Deleted
	Renamed
	Copied
	Untracked
)

func (s ChangeStatus) String() string {
	switch s {
	case Modified:
		return "M"
	case Added:
		return "A"
	case Deleted:
		return "D"
	case Renamed:
		return "R"
	case Copied:
		return "C"
	case Untracked:
		return "?"
	}
	return "?"
}

type LineType int

const (
	Context LineType = iota
	LineAdded
	LineRemoved
	Header
)

type DiffLine struct {
	Content string
	Type    LineType
}

type FileChange struct {
	Path       string
	Status     ChangeStatus
	Staged     bool
	Unstaged   bool
	Expanded   bool
	DiffLines  []DiffLine
	DiffLoaded bool
	Additions  int
	Deletions  int
}

type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterUnstaged
	FilterStaged
)

func (f FilterMode) String() string {
	switch f {
	case FilterAll:
		return "all"
	case FilterUnstaged:
		return "unstaged"
	case FilterStaged:
		return "staged"
	}
	return "all"
}
