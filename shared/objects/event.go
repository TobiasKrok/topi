package objects

type BuildTrigger struct {
	Repository string
	Ref        string
	Commit     BuildCommit
	Timestamp  int64
}
type BuildCommit struct {
	Sha       string
	Message   string
	Committer string
}
