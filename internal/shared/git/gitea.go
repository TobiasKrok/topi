package git

import (
	"code.gitea.io/sdk/gitea"
	"fmt"
	"os"
)

type GiteaService struct {
	client *gitea.Client
}

func NewGiteaService() *GiteaService {

	gitea.SetToken(os.Getenv("GITEA_TOKEN"))
	client, err := gitea.NewClient(os.Getenv("GITEA_HOST"))
	if err != nil {
		panic(err)
	}
	return &GiteaService{
		client: client,
	}
}

func (g *GiteaService) GetFile(owner, name, ref, filepath string) ([]byte, error) {
	data, res, err := g.client.GetFile(owner, name, ref, filepath)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return data, nil

}

func (g *GiteaService) GetTree(owner string, name string, ref string, recursive bool) (GitTree, error) {

	t, res, err := g.client.GetTrees(owner, name, gitea.ListTreeOptions{Recursive: recursive, Ref: ref})
	if err != nil {
		return GitTree{}, err
	}
	if res.StatusCode != 200 {
		return GitTree{}, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	var tree GitTree
	var treeEntries []TreeEntry

	tree.Path = t.URL
	tree.TotalCount = t.TotalCount
	tree.Entries = treeEntries
	for _, entry := range t.Entries {
		treeEntries = append(treeEntries, TreeEntry{
			Path: entry.Path,
			Type: entry.Type,
			Mode: entry.Mode,
			Size: entry.Size,
			Sha:  entry.SHA,
		})
	}
	return tree, nil
}

//func (g *giteaService) GetRepo(owner string, name string) error {
//	repo, res, err := g.client.GetRepo(owner, name)
//	if err != nil {
//		return err
//	}
//	if res.StatusCode != 200 {
//		return nil
//	}
//
//	return nil
//}
