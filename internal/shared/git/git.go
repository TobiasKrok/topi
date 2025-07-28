package git

type GitService interface {
	GetFile(owner, name, ref, filepath string) ([]byte, error)
	GetTree(owner, repo string) (GitTree, error)
}

type GitTree struct {
	Path    string
	Entries []TreeEntry
}

type TreeEntry struct {
	Path string
	Type string
	Mode string
	Size int64
	Sha  string
	URL  string
}
