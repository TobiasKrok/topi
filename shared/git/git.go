package git

// ref can be branch, tag, commit
type GitService interface {
	GetFile(owner, name, ref, filepath string) ([]byte, error)
	GetTree(owner, repo, ref string, recursive bool) (GitTree, error)
}

type GitTree struct {
	Path       string
	Entries    []TreeEntry
	TotalCount int
}

type TreeEntry struct {
	Path string
	Type string
	Mode string
	Size int64
	Sha  string
	URL  string
}
